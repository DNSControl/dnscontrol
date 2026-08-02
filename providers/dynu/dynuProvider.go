// Package dynu implements a DNSControl provider for Dynu (https://www.dynu.com).
// API docs: https://www.dynu.com/en-US/Resources/API
// Auth: set api_key in creds.json.
// Module: github.com/DNSControl/dnscontrol/v5
package dynu

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

var features = providers.DocumentationNotes{
	providers.CanGetZones:      providers.Can(),
	providers.CanConcur:        providers.Cannot(),
	providers.CanUseSRV:        providers.Can(),
	providers.CanUseCAA:        providers.Can(),
	providers.CanUseDNAME:      providers.Can(),
	providers.CanUseDHCID:      providers.Can(),
	providers.CanUseHTTPS:      providers.Can(),
	providers.CanUseLOC:        providers.Can(),
	providers.CanUseNAPTR:      providers.Can(),
	providers.CanUseOPENPGPKEY: providers.Can(),
	providers.CanUsePTR:        providers.Can(),
	providers.CanUseRP:         providers.Can(),
	providers.CanUseSMIMEA:     providers.Can(),
	providers.CanUseSVCB:       providers.Can(),
	providers.CanUseTLSA:       providers.Can(),
	providers.CanUseSSHFP:      providers.Can(),
	providers.CanUseAlias:      providers.Cannot(),
	providers.CanAutoDNSSEC:    providers.Cannot(),
}

func init() {
	fns := providers.DspFuncs{
		Initializer:   New,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType("DYNU", fns, features)
	providers.RegisterCredsMetadata("DYNU", providers.CredsMetadata{
		DisplayName: "Dynu",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/dynu",
		PortalURL:   "https://www.dynu.com/en-US/ControlPanel",
		Fields: []providers.CredsField{
			{Key: "api_key", Label: "API Key", Required: true, Secret: true},
		},
	})
}

// New creates a Dynu provider from credentials.
func New(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	apiKey := m["api_key"]
	if apiKey == "" {
		return nil, fmt.Errorf("missing Dynu API key")
	}
	return &dynuProvider{
		apiKey:    apiKey,
		domainIDs: map[string]int64{},
	}, nil
}

// GetNameservers returns Dynu's authoritative nameservers.
func (d *dynuProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers([]string{
		"ns1.dynu.com",
		"ns2.dynu.com",
		"ns3.dynu.com",
		"ns4.dynu.com",
		"ns5.dynu.com",
		"ns6.dynu.com",
	})
}

// GetZoneRecords downloads all records for the zone and returns them as RecordConfigs.
func (d *dynuProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domainID, err := d.getDomainID(dc.Name)
	if err != nil {
		return nil, err
	}
	records, err := d.getRecords(domainID)
	if err != nil {
		return nil, err
	}
	var existing models.Records
	for _, r := range records {
		rc, err := toRc(r, dc)
		if err != nil {
			return nil, err
		}
		if rc != nil {
			existing = append(existing, rc)
		}
	}
	return existing, nil
}

// GetZoneRecordsCorrections computes the corrections needed to bring the zone to the desired state.
func (d *dynuProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	domainID, err := d.getDomainID(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	instructions, _, err := diff2.ByRecord(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, inst := range instructions {
		// Apex NS records are managed by Dynu internally and cannot be created,
		// modified, or deleted via the API.
		if len(inst.New) > 0 && inst.New[0].Type == "NS" && inst.New[0].Name == "@" {
			continue
		}
		if len(inst.Old) > 0 && inst.Old[0].Type == "NS" && inst.Old[0].Name == "@" {
			continue
		}

		switch inst.Type {
		case diff2.CREATE:
			req := toReq(inst.New[0])
			msg := strings.Join(inst.Msgs, "\n")
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					return d.createRecord(domainID, req)
				},
			})
		case diff2.CHANGE:
			// Dynu overrides NS record TTL to 3600 and does not allow modifying NS content.
			// Silently skip CHANGE corrections for NS records to maintain idempotency.
			if inst.New[0].Type == "NS" {
				continue
			}
			req := toReq(inst.New[0])
			oldID := inst.Old[0].Original.(*dynuRecord).ID
			msg := strings.Join(inst.Msgs, "\n")
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					return d.updateRecord(domainID, oldID, req)
				},
			})
		case diff2.DELETE:
			oldID := inst.Old[0].Original.(*dynuRecord).ID
			msg := strings.Join(inst.Msgs, "\n")
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					return d.deleteRecord(domainID, oldID)
				},
			})
		}
	}
	return corrections, len(corrections), nil
}

// GetZones returns all DNS zones in the account (implements providers.ZoneLister).
func (d *dynuProvider) GetZones() ([]string, error) {
	domains, err := d.getDomains()
	if err != nil {
		return nil, err
	}
	zones := make([]string, len(domains))
	for i, dom := range domains {
		zones[i] = dom.Name
	}
	return zones, nil
}

// toRc converts a Dynu API record to a DNSControl RecordConfig.
// Returns (nil, nil) for record types managed internally by Dynu (SOA, WCA).
// NOTE: r.Content from the Dynu API is the full zone-file line (hostname TTL class
// type rdata), not just the rdata. We therefore always use the individual structured
// fields returned by Dynu rather than r.Content.
func toRc(r *dynuRecord, dc *models.DomainConfig) (*models.RecordConfig, error) {
	switch r.RecordType {
	case "SOA", "WCA":
		return nil, nil
	}

	domain := dc.Name
	var rc *models.RecordConfig
	var err error
	label := dc.LabelFromShort(r.NodeName)
	ttl := uint32(r.TTL)
	switch r.RecordType {
	case "A":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeA, r.IPv4Address)
	case "AAAA":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeAAAA, r.IPv6Address)
	case "AFSDB":
		rc, err = dc.NewRecordConfigParse(label, ttl, dnsv2.TypeAFSDB, fmt.Sprintf("%d %s", intOrZero(r.SubType), ensureTrailingDot(r.Host)))
	case "CAA":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeCAA, intOrZero(r.Flags), r.Tag, r.Value)
	case "CERT":
		rc, err = dc.NewRecordConfigParse(label, ttl, dnsv2.TypeCERT, fmt.Sprintf("%d %d %d %s", intOrZero(r.CertificateType), intOrZero(r.KeyTag), intOrZero(r.Algorithm), r.Certificate))
	case "CNAME":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeCNAME, ensureTrailingDot(r.Host))
	case "DHCID":
		rc, err = dc.NewRecordConfigParse(label, ttl, dnsv2.TypeDHCID, r.RecordData)
	case "DNAME":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeDNAME, ensureTrailingDot(r.Host))
	case "HINFO":
		rc, err = dc.NewRecordConfigParse(label, ttl, dnsv2.TypeHINFO, fmt.Sprintf("%q %q", r.CPU, r.OperatingSystem))
	case "HTTPS", "SVCB":
		// Build rdata from individual structured Dynu fields.
		// targetName is returned by Dynu with a trailing dot already.
		target := r.TargetName
		if target == "" {
			target = "."
		}
		rdata := fmt.Sprintf("%d %s", intOrZero(r.SvcPriority), target) // ignore:legacyfield
		if ps := svcParamsToString(r.SvcParams); ps != "" {             // ignore:legacyfield
			rdata += " " + ps
		}
		rc, err = dc.NewRecordConfigParse(label, ttl, r.RecordType, rdata)
	case "KEY":
		rc, err = dc.NewRecordConfigParse(label, ttl, dnsv2.TypeKEY, fmt.Sprintf("%d %d %d %s", intOrZero(r.Flags), intOrZero(r.KeyProtocol), intOrZero(r.Algorithm), r.PublicKey))
	case "LOC":
		// Parse DMS components from the content string (avoids miekg precision issues).
		// Use Dynu's individual metric fields (r.Altitude, r.Size, r.Horizontal/
		// VerticalPrecision) for the metre values — they are always present in the
		// API response and more reliable than counting fields in the content string.
		rdata := extractRdata(r.Content, "LOC")
		if rdata == "" {
			return nil, fmt.Errorf("LOC record missing content for %s", r.Hostname)
		}
		d1, m1, s1, ns, d2, m2, s2, ew, al, sz, hp, vp, locErr := parseLOCRdata(rdata)
		if locErr != nil {
			return nil, fmt.Errorf("LOC rdata parse error for %s: %w", r.Hostname, locErr)
		}
		// Override metric values with the structured API response fields, which are
		// always populated by Dynu even when the content string omits trailing zeros.
		if r.Altitude != nil {
			al = float32(*r.Altitude)
		}
		if r.Size != nil {
			sz = float32(*r.Size)
		}
		if r.HorizontalPrecision != nil {
			hp = float32(*r.HorizontalPrecision)
		}
		if r.VerticalPrecision != nil {
			vp = float32(*r.VerticalPrecision)
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeLOC, d1, m1, s1, ns, d2, m2, s2, ew, float64(al), sz, hp, vp)
	case "MX":
		host := r.Host
		// Dynu stores null MX (priority 0, target ".") by returning the zone name as host.
		if intOrZero(r.Priority) == 0 && (host == "" || strings.TrimSuffix(host, ".") == domain) {
			host = "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, intOrZero(r.Priority), ensureTrailingDot(host))
	case "NAPTR":
		// Dynu stores the null replacement (".") as an empty string.
		naptrReplacement := r.Replacement
		if naptrReplacement == "" {
			naptrReplacement = "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeNAPTR,
			intOrZero(r.Order), intOrZero(r.Preference), r.NaptrFlags, r.Services, r.RegExp, ensureTrailingDot(naptrReplacement)) // ignore:legacyfield
	case "NS":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeNS, ensureTrailingDot(r.Host))
	case "OPENPGPKEY":
		rc, err = dc.NewRecordConfigParse(label, ttl, dnsv2.TypeOPENPGPKEY, r.PublicKey)
	case "PTR":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypePTR, ensureTrailingDot(r.Host))
	case "RP":
		mbox := ensureTrailingDot(r.MailBox)
		txt := ensureTrailingDot(r.TxtDomainName)
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeRP, mbox, txt)
	case "SMIMEA":
		certHex, convErr := base64ToHex(r.CertificateAssociatedData)
		if convErr != nil {
			return nil, fmt.Errorf("SMIMEA certAssocData base64 decode for %s: %w", r.Hostname, convErr)
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSMIMEA, intOrZero(r.CertificateUsage), intOrZero(r.Selector), intOrZero(r.MatchingType), certHex)
	case "SPF", "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, r.TextData)
	case "SRV":
		// Dynu stores the null SRV target (".") as an empty host string.
		srvHost := r.Host
		if srvHost == "" {
			srvHost = "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSRV, intOrZero(r.Priority), intOrZero(r.Weight), intOrZero(r.Port), ensureTrailingDot(srvHost))
	case "SSHFP":
		fpHex, convErr := base64ToHex(r.FingerPrint)
		if convErr != nil {
			return nil, fmt.Errorf("SSHFP fingerprint base64 decode for %s: %w", r.Hostname, convErr)
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSSHFP, intOrZero(r.Algorithm), intOrZero(r.FingerPrintType), fpHex)
	case "TLSA":
		certHex, convErr := base64ToHex(r.CertificateAssociatedData)
		if convErr != nil {
			return nil, fmt.Errorf("TLSA certAssocData base64 decode for %s: %w", r.Hostname, convErr)
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTLSA, intOrZero(r.CertificateUsage), intOrZero(r.Selector), intOrZero(r.MatchingType), certHex)
	case "URI":
		rc, err = dc.NewRecordConfigParse(label, ttl, dnsv2.TypeURI, fmt.Sprintf("%d %d %q", intOrZero(r.Priority), intOrZero(r.Weight), r.TargetURI))
	default:
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("record %s %s: %w", r.RecordType, r.Hostname, err)
	}
	rc.Original = r
	return rc, nil
}

// toReq converts a DNSControl RecordConfig to a Dynu API create/update request body.
func toReq(rc *models.RecordConfig) *dynuRecord {
	nodeName := rc.Name
	if nodeName == "@" {
		nodeName = ""
	}
	req := &dynuRecord{
		NodeName:   nodeName,
		RecordType: rc.Type,
		TTL:        int(rc.TTL),
		State:      true,
	}
	switch rc.Type {
	case "A":
		req.IPv4Address = rc.AsA().String()
	case "AAAA":
		req.IPv6Address = rc.AsAAAA().String()
	case "AFSDB":
		// Target: "<subtype> <hostname>."

		// parts := strings.Fields(rc.GetTargetField())
		// if len(parts) >= 2 {
		// 	st, _ := strconv.Atoi(parts[0])
		// 	req.SubType = &st
		// 	req.Host = strings.TrimSuffix(parts[1], ".")
		// }

		f := rc.AsAFSDB()
		req.SubType = new(int(f.Subtype))
		req.Host = strings.TrimSuffix(f.Hostname, ".")
	case "CAA":
		f := rc.AsCAA()
		flags := int(f.Flag)
		req.Flags = &flags
		req.Tag = f.Tag
		req.Value = f.Value
	case "CERT":

		// Target: "<type> <keytag> <algorithm> <cert-base64>"
		// parts := strings.Fields(rc.GetTargetField())
		// if len(parts) >= 4 {
		// 	ct := parseCERTType(parts[0])
		// 	kt, _ := strconv.Atoi(parts[1])
		// 	algo, _ := strconv.Atoi(parts[2])
		// 	req.CertificateType = &ct
		// 	req.KeyTag = &kt
		// 	req.Algorithm = &algo
		// 	req.Certificate = parts[3]
		// }

		f := rc.AsCERT()
		req.CertificateType = new(int(f.Type))
		req.KeyTag = new(int(f.KeyTag))
		req.Algorithm = new(int(f.Algorithm))
		req.Certificate = f.Certificate

	case "CNAME", "NS", "PTR", "DNAME":
		req.Host = strings.TrimSuffix(rc.GetRDATA().String(), ".")
	case "DHCID":
		// Target is the base64-encoded DHCID data (zone-file format == API format).
		req.RecordData = rc.AsDHCID().Digest
	case "HINFO":

		// Target: "<"cpu"> <"os">" — parse the two quoted character-strings.
		// cpu, os := parseCharStrings(rc.GetTargetField())
		// req.CPU = cpu
		// req.OperatingSystem = os

		req.CPU = rc.AsHINFO().Cpu
		req.OperatingSystem = rc.AsHINFO().Os
	case "HTTPS":
		f := rc.AsHTTPS()
		req.SvcPriority = new(int(f.Priority)) // ignore:legacyfield
		// Preserve "." for the null target; strip trailing dot from real hostnames.
		target := strings.TrimSuffix(f.Target, ".")
		if target == "" {
			target = "."
		}
		req.TargetName = target
		req.SvcParams = parseSvcParams(models.Svcbv2ValueToString(f.Value)) // ignore:legacyfield
	case "KEY":
		// Target: "<flags> <protocol> <algorithm> <pubkey-base64>"

		// parts := strings.Fields(rc.GetTargetField())
		// if len(parts) >= 4 {
		// 	f, _ := strconv.Atoi(parts[0])
		// 	proto, _ := strconv.Atoi(parts[1])
		// 	algo, _ := strconv.Atoi(parts[2])
		// 	req.Flags = &f
		// 	req.KeyProtocol = &proto
		// 	req.Algorithm = &algo
		// 	req.PublicKey = parts[3]
		// }

		f := rc.AsKEY()
		req.Flags = new(int(f.Flags))
		req.KeyProtocol = new(int(f.Protocol))
		req.Algorithm = new(int(f.Algorithm))
		req.PublicKey = f.PublicKey

	case "LOC":
		// Convert DNSControl's packed binary LOC fields to Dynu's decimal-degree format.
		// The packed values are integer arc-milliseconds. We compute total ms first
		// (avoiding intermediate fractional divisions), then add a +0.5 ms bias before
		// dividing. This ensures Dynu's internal floor-based DMS conversion always
		// rounds to the correct integer millisecond (e.g. 71°06'18.000" rather than
		// 71°06'17.999" when the float64 representation of 71.105 is slightly below exact).
		const locMsPerDegree = 3600000.0
		f := rc.AsLOC()
		latHemi, latDeg, latMin, latSec := models.ReverseLatitude(f.Latitude)
		latMs := float64(latDeg)*locMsPerDegree + float64(latMin)*60000 + latSec*1000
		lat := (latMs + 0.5) / locMsPerDegree
		if latHemi == "S" {
			lat = -lat
		}
		lonHemi, lonDeg, lonMin, lonSec := models.ReverseLongitude(f.Longitude)
		lonMs := float64(lonDeg)*locMsPerDegree + float64(lonMin)*60000 + lonSec*1000
		lon := (lonMs + 0.5) / locMsPerDegree
		if lonHemi == "W" {
			lon = -lon
		}
		alt := models.ReverseAltitude(f.Altitude)
		size := models.ReverseENotationInt(f.Size)
		horizPre := models.ReverseENotationInt(f.HorizPre)
		vertPre := models.ReverseENotationInt(f.VertPre)
		req.Latitude = &lat
		req.Longitude = &lon
		req.Altitude = &alt
		req.Size = &size
		req.HorizontalPrecision = &horizPre
		req.VerticalPrecision = &vertPre
	case "MX":
		f := rc.AsMX()
		req.Host = strings.TrimSuffix(f.Mx, ".")
		pref := int(f.Preference)
		req.Priority = &pref
	case "NAPTR":
		f := rc.AsNAPTR()
		order := int(f.Order)
		pref := int(f.Preference)
		req.Order = &order
		req.Preference = &pref
		req.NaptrFlags = f.Flags // ignore:legacyfield
		req.Services = f.Service
		req.RegExp = f.Regexp
		// Preserve "." as-is (null replacement); strip trailing dot from real FQDNs.
		naptrTarget := f.Service
		if naptrTarget != "." {
			naptrTarget = strings.TrimSuffix(naptrTarget, ".")
		}
		req.Replacement = naptrTarget
	case "OPENPGPKEY":
		// Target is the base64-encoded public key (zone-file format == API format).
		req.PublicKey = rc.AsOPENPGPKEY().PublicKey
	case "RP":
		rd := rc.AsRP()
		req.MailBox = rd.Mbox
		req.TxtDomainName = rd.Txt
	case "SMIMEA":

		// usage := int(rc.SmimeaUsage)
		// selector := int(rc.SmimeaSelector)
		// mtype := int(rc.SmimeaMatchingType)
		// req.CertificateUsage = &usage
		// req.Selector = &selector
		// req.MatchingType = &mtype
		// req.CertificateAssociatedData = hexToBase64(rc.GetTargetField())

		f := rc.AsSMIMEA()
		usage := int(f.Usage)
		selector := int(f.Selector)
		mtype := int(f.MatchingType)
		req.CertificateUsage = &usage
		req.Selector = &selector
		req.MatchingType = &mtype
		req.CertificateAssociatedData = hexToBase64(f.Certificate)
	case "SRV":
		// Preserve "." for the null target; strip trailing dot from real hostnames.
		f := rc.AsSRV()
		srvTarget := strings.TrimSuffix(f.Target, ".")
		if srvTarget == "" {
			srvTarget = "."
		}
		req.Host = srvTarget
		prio := int(f.Priority)
		weight := int(f.Weight)
		port := int(f.Port)
		req.Priority = &prio
		req.Weight = &weight
		req.Port = &port
	case "SSHFP":
		f := rc.AsSSHFP()
		algo := int(f.Algorithm)
		fptype := int(f.Type)
		req.Algorithm = &algo
		req.FingerPrintType = &fptype
		req.FingerPrint = hexToBase64(f.FingerPrint)
	case "SVCB":
		f := rc.AsSVCB()
		req.SvcPriority = new(int(f.Priority)) // ignore:legacyfield
		target := strings.TrimSuffix(f.Target, ".")
		if target == "" {
			target = "."
		}
		req.TargetName = target
		req.SvcParams = parseSvcParams(models.Svcbv2ValueToString(f.Value)) // ignore:legacyfield
	case "TLSA":
		f := rc.AsTLSA()
		usage := int(f.Usage)
		selector := int(f.Selector)
		mtype := int(f.MatchingType)
		req.CertificateUsage = &usage
		req.Selector = &selector
		req.MatchingType = &mtype
		req.CertificateAssociatedData = hexToBase64(f.Certificate)
	case "TXT":
		req.TextData = rc.GetTargetTXTJoined()
	case "URI":
		// Target: "<priority> <weight> "<target-uri>""

		// parts := strings.SplitN(strings.TrimSpace(rc.GetTargetField()), " ", 3)
		// if len(parts) >= 3 {
		// 	prio, _ := strconv.Atoi(parts[0])
		// 	wgt, _ := strconv.Atoi(parts[1])
		// 	req.Priority = &prio
		// 	req.Weight = &wgt
		// 	req.TargetURI = strings.Trim(parts[2], "\"")
		// }

		f := rc.AsURI()
		req.Priority = new(int(f.Priority))
		req.Weight = new(int(f.Weight))
		req.TargetURI = strings.Trim(f.Target, "\"")

	}
	return req
}

// parseSvcParams converts DNSControl's space-separated SvcParams string (e.g. "alpn=h2,h3 port=443")
// into the slice of typed objects that the Dynu API expects.
func parseSvcParams(s string) []svcParam {
	if s == "" {
		return nil
	}
	var result []svcParam
	for part := range strings.FieldsSeq(s) {
		kv := strings.SplitN(part, "=", 2)
		key := strings.ToLower(kv[0])
		val := ""
		if len(kv) == 2 {
			val = kv[1]
		}
		switch key {
		case "alpn":
			result = append(result, svcParam{Type: "ALPN", AlpnIds: strings.Split(val, ",")})
		case "no-default-alpn":
			result = append(result, svcParam{Type: "NoDefaultALPN"})
		case "port":
			p, _ := strconv.Atoi(val)
			sp := svcParam{Type: "Port", Port: &p}
			result = append(result, sp)
		case "ipv4hint":
			result = append(result, svcParam{Type: "IPv4Hint", IPv4Hints: strings.Split(val, ",")})
		case "ipv6hint":
			result = append(result, svcParam{Type: "IPv6Hint", IPv6Hints: strings.Split(val, ",")})
		case "mandatory":
			result = append(result, svcParam{Type: "Mandatory", Keys: strings.Split(val, ",")})
		case "ech":
			result = append(result, svcParam{Type: "ECH", ECH: val})
		}
	}
	return result
}

// svcParamsToString converts Dynu's typed []svcParam slice to DNSControl's
// space-separated SvcParams string (e.g. "alpn=h2,h3 port=443").
func svcParamsToString(params []svcParam) string {
	var parts []string
	for _, p := range params {
		switch strings.ToUpper(p.Type) {
		case "ALPN":
			parts = append(parts, "alpn="+strings.Join(p.AlpnIds, ","))
		case "NODEFAULTALPN":
			parts = append(parts, "no-default-alpn")
		case "PORT":
			if p.Port != nil {
				parts = append(parts, fmt.Sprintf("port=%d", *p.Port))
			}
		case "IPV4HINT":
			parts = append(parts, "ipv4hint="+strings.Join(p.IPv4Hints, ","))
		case "IPV6HINT":
			parts = append(parts, "ipv6hint="+strings.Join(p.IPv6Hints, ","))
		case "MANDATORY":
			parts = append(parts, "mandatory="+strings.Join(p.Keys, ","))
		case "ECH":
			if p.ECH != "" {
				parts = append(parts, "ech="+p.ECH)
			}
		}
	}
	return strings.Join(parts, " ")
}

// // locRdata builds the LOC rdata string (rdata-only, no owner/TTL) from Dynu's
// // decimal-degree and metre fields so it can be passed to SetTargetLOCString.
// func locRdata(lat, lon, alt, size, horizPre, vertPre float64) string {
// 	latD, latM, latS, latHemi := ddToDMS(lat, "N", "S")
// 	lonD, lonM, lonS, lonHemi := ddToDMS(lon, "E", "W")
// 	return fmt.Sprintf("%d %d %.3f %s %d %d %.3f %s %.2fm %.2fm %.2fm %.2fm",
// 		latD, latM, latS, latHemi, lonD, lonM, lonS, lonHemi,
// 		alt, size, horizPre, vertPre)
// }

// // ddToDMS converts a signed decimal-degrees value to unsigned degrees, minutes,
// // seconds, and hemisphere strings (pos/neg for the two possible hemispheres).
// func ddToDMS(dd float64, pos, neg string) (d uint8, m uint8, s float32, hemi string) {
// 	hemi = pos
// 	if dd < 0 {
// 		hemi = neg
// 		dd = -dd
// 	}
// 	d = uint8(dd)
// 	minutesTotal := (dd - float64(d)) * 60.0
// 	m = uint8(minutesTotal)
// 	s = float32((minutesTotal - float64(m)) * 60.0)
// 	return
// }

// // floatOrDefault returns *f if non-nil, otherwise the supplied default.
// func floatOrDefault(f *float64, def float64) float64 {
// 	if f == nil {
// 		return def
// 	}
// 	return *f
// }

// extractRdata strips the owner name, TTL, optional class, and type from a full
// DNS zone-file line ("hostname. TTL [IN] TYPE rdata") and returns just the rdata.
// Returns "" if rtype is not found.
func extractRdata(content, rtype string) string {
	fields := strings.Fields(content)
	for i, f := range fields {
		if strings.EqualFold(f, rtype) && i < len(fields)-1 {
			return strings.Join(fields[i+1:], " ")
		}
	}
	return ""
}

// // parseCharStrings splits a string containing one or two DNS character-strings
// // (optionally quoted) and returns the first two values.
// // Used for HINFO: "X86-64" "Linux" → ("X86-64", "Linux").
// func parseCharStrings(s string) (first, second string) {
// 	s = strings.TrimSpace(s)
// 	var parts []string
// 	for len(s) > 0 {
// 		s = strings.TrimLeft(s, " \t")
// 		if len(s) == 0 {
// 			break
// 		}
// 		if s[0] == '"' {
// 			end := strings.Index(s[1:], "\"")
// 			if end < 0 {
// 				parts = append(parts, s[1:])
// 				break
// 			}
// 			parts = append(parts, s[1:end+1])
// 			s = s[end+2:]
// 		} else {
// 			idx := strings.IndexAny(s, " \t")
// 			if idx < 0 {
// 				parts = append(parts, s)
// 				break
// 			}
// 			parts = append(parts, s[:idx])
// 			s = s[idx:]
// 		}
// 	}
// 	if len(parts) >= 1 {
// 		first = parts[0]
// 	}
// 	if len(parts) >= 2 {
// 		second = parts[1]
// 	}
// 	return
// }

// // parseCERTType converts a CERT type name or number string to its integer value.
// func parseCERTType(s string) int {
// 	named := map[string]int{
// 		"PKIX": 1, "SPKI": 2, "PGP": 3, "IPKIX": 4,
// 		"ISPKI": 5, "IPGP": 6, "ACPKIX": 7, "IACPKIX": 8,
// 		"URI": 253, "OID": 254,
// 	}
// 	if v, ok := named[strings.ToUpper(s)]; ok {
// 		return v
// 	}
// 	v, _ := strconv.Atoi(s)
// 	return v
// }

// parseLOCRdata parses a LOC record rdata string (the part after the type label)
// into the 12 parameters required by models.RecordConfig.SetLOCParams.
// It uses strings.Fields so it is robust to Dynu's variable-precision output
// (e.g. "0m" vs "0.00m", integer vs decimal seconds).
func parseLOCRdata(rdata string) (d1, m1 uint8, s1 float32, ns string, d2, m2 uint8, s2 float32, ew string, al, sz, hp, vp float32, err error) {
	fields := strings.Fields(rdata)
	if len(fields) < 8 {
		err = fmt.Errorf("too few fields (%d)", len(fields))
		return
	}
	parseUint8 := func(s string) (uint8, error) {
		var v int
		if _, e := fmt.Sscanf(s, "%d", &v); e != nil {
			return 0, e
		}
		return uint8(v), nil
	}
	parseF32 := func(s string) (float32, error) {
		var v float64
		if _, e := fmt.Sscanf(s, "%f", &v); e != nil {
			return 0, e
		}
		return float32(v), nil
	}
	if d1, err = parseUint8(fields[0]); err != nil {
		return
	}
	if m1, err = parseUint8(fields[1]); err != nil {
		return
	}
	if s1, err = parseF32(fields[2]); err != nil {
		return
	}
	ns = fields[3]
	if d2, err = parseUint8(fields[4]); err != nil {
		return
	}
	if m2, err = parseUint8(fields[5]); err != nil {
		return
	}
	if s2, err = parseF32(fields[6]); err != nil {
		return
	}
	ew = fields[7]
	// Remaining fields: alt, size, horizPre, vertPre — each suffixed with "m".
	parseMetres := func(s string) (float32, error) {
		return parseF32(strings.TrimSuffix(s, "m"))
	}
	if len(fields) > 8 {
		if al, err = parseMetres(fields[8]); err != nil {
			return
		}
	}
	if len(fields) > 9 {
		if sz, err = parseMetres(fields[9]); err != nil {
			return
		}
	}
	if len(fields) > 10 {
		if hp, err = parseMetres(fields[10]); err != nil {
			return
		}
	}
	if len(fields) > 11 {
		vp, err = parseMetres(fields[11])
	}
	return
}

// base64ToHex converts a base64-encoded string to its lowercase hex representation.
func base64ToHex(b64 string) (string, error) {
	if b64 == "" {
		return "", nil
	}
	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hexToBase64 converts a hex string to standard base64.
// Returns the input unchanged if it cannot be hex-decoded (defensive fallback).
func hexToBase64(hexStr string) string {
	if hexStr == "" {
		return ""
	}
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return hexStr
	}
	return base64.StdEncoding.EncodeToString(b)
}

func intOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func ensureTrailingDot(s string) string {
	if s == "" || strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}
