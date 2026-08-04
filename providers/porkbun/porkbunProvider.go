package porkbun

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

const (
	minimumTTL = 600
)

const (
	metaType        = "type"
	metaIncludePath = "includePath"
	metaWildcard    = "wildcard"
)

const (
	defaultMaxAttempts = 5
)

// https://kb.porkbun.com/article/63-how-to-switch-to-porkbuns-nameservers
var defaultNS = []string{
	"curitiba.ns.porkbun.com",
	"fortaleza.ns.porkbun.com",
	"maceio.ns.porkbun.com",
	"salvador.ns.porkbun.com",
}

func newReg(conf map[string]string) (providers.Registrar, error) {
	return newPorkbun(conf, nil)
}

func newDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newPorkbun(conf, metadata)
}

// newPorkbun creates the provider.
func newPorkbun(m map[string]string, _ json.RawMessage) (*porkbunProvider, error) {
	c := &porkbunProvider{
		maxAttempts: defaultMaxAttempts,
	}

	c.apiKey, c.secretKey = m["api_key"], m["secret_key"]

	if c.apiKey == "" || c.secretKey == "" {
		return nil, errors.New("missing porkbun api_key or secret_key")
	}

	if maxAttempts, ok := m["max_attempts"]; ok && maxAttempts != "" {
		i, err := strconv.Atoi(maxAttempts)
		if err != nil {
			return nil, fmt.Errorf("porkbun: invalid max_attempts %q: must be a whole number", maxAttempts)
		}
		c.maxAttempts = i
	}
	if maxDuration, ok := m["max_duration"]; ok && maxDuration != "" {
		d, err := time.ParseDuration(maxDuration)
		if err != nil {
			return nil, fmt.Errorf("porkbun: invalid max_duration %q: valid units are ns, us, ms, s, m, h", maxDuration)
		}
		c.maxDuration = d
	}

	return c, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.DocCreateDomains:       providers.Cannot(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "PORKBUN"
	const providerMaintainer = "@imlonghao"
	providers.RegisterRegistrarType(providerName, newReg)
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Porkbun",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/porkbun",
		PortalURL:   "https://porkbun.com/account/api",
		Notes:       "Porkbun requires API access to be enabled for each domain before DNSControl can manage it.",
		Fields: []providers.CredsField{
			{
				Key:      "api_key",
				Label:    "API key",
				Help:     "The API Key generated from Porkbun API Access.",
				Secret:   true,
				Required: true,
			},
			{
				Key:      "secret_key",
				Label:    "Secret key",
				Help:     "The Secret Key shown when creating the Porkbun API key.",
				Secret:   true,
				Required: true,
			},
			{
				Key:   "max_attempts",
				Label: "Max attempts (optional)",
				Help:  "Override retry attempts. Leave blank to use the default of 5.",
			},
			{
				Key:   "max_duration",
				Label: "Max duration (optional)",
				Help:  "Retry duration limit, such as 5m. Leave blank for no limit.",
			},
		},
	})
	providers.RegisterCustomRecordType("PORKBUN_URLFWD", providerName, "")
	providers.RegisterCustomRecordType("URL", providerName, "")
	providers.RegisterCustomRecordType("URL301", providerName, "")
}

// GetNameservers returns the nameservers for a domain.
func (c *porkbunProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(defaultNS)
}

// isURLForwardingType returns true if the record type is a URL forwarding type.
func isURLForwardingType(recordType string) bool {
	return recordType == "PORKBUN_URLFWD" || recordType == "URL" || recordType == "URL301"
}

func genComparable(rec *models.RecordConfig) string {
	if isURLForwardingType(rec.Type) {
		return fmt.Sprintf("includePath=%s wildcard=%s", rec.Metadata[metaIncludePath], rec.Metadata[metaWildcard])
	}
	return ""
}

func porkbunURLForwardingMetadata(recordType string, metadata map[string]string) (string, string, string) {
	t := metadata[metaType]
	if t == "" {
		if recordType == "URL301" {
			t = "permanent"
		} else {
			t = "temporary"
		}
	}
	includePath := metadata[metaIncludePath]
	if includePath == "" {
		includePath = "no"
	}
	wildcard := metadata[metaWildcard]
	if wildcard == "" {
		wildcard = "yes"
	}
	return t, includePath, wildcard
}

func normalizeURLForwardingRecord(record *models.RecordConfig, _ string) {
	if !isURLForwardingType(record.Type) {
		return
	}
	record.TTL = 0
	if record.Metadata == nil {
		record.Metadata = make(map[string]string)
	}
	t, includePath, wildcard := porkbunURLForwardingMetadata(record.Type, record.Metadata)
	record.Metadata[metaType] = t
	record.Metadata[metaIncludePath] = includePath
	record.Metadata[metaWildcard] = wildcard
	if record.Type == "PORKBUN_URLFWD" {
		if record.Metadata[metaType] == "permanent" {
			record.Type = "URL301"
		} else {
			record.Type = "URL"
		}
	}
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *porkbunProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	// Block changes to NS records for base domain
	checkNSModifications(dc)

	// Make sure TTL larger than the minimum TTL
	for _, record := range dc.Records {
		record.TTL = fixTTL(record.TTL)
		normalizeURLForwardingRecord(record, dc.Name)
	}

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, genComparable)
	if err != nil {
		return nil, 0, err
	}
	for _, change := range changes {
		var corr *models.Correction
		switch change.Type {
		case diff2.REPORT:
			corr = &models.Correction{Msg: change.MsgsJoined}
		case diff2.CREATE:
			req, err := toReq(change.New[0])
			if err != nil {
				return nil, 0, err
			}
			corr = &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					if isURLForwardingType(change.New[0].Type) {
						return c.createURLForwardingRecord(dc.Name, req)
					}
					return c.createRecord(dc.Name, req)
				},
			}
		case diff2.CHANGE:
			id := change.Old[0].Original.(*domainRecord).ID
			req, err := toReq(change.New[0])
			if err != nil {
				return nil, 0, err
			}
			corr = &models.Correction{
				Msg: fmt.Sprintf("%s, porkbun ID: %s", change.Msgs[0], id),
				F: func() error {
					if isURLForwardingType(change.New[0].Type) {
						return c.modifyURLForwardingRecord(dc.Name, id, req)
					}
					return c.modifyRecord(dc.Name, id, req)
				},
			}
		case diff2.DELETE:
			id := change.Old[0].Original.(*domainRecord).ID
			corr = &models.Correction{
				Msg: fmt.Sprintf("%s, porkbun ID: %s", change.Msgs[0], id),
				F: func() error {
					if isURLForwardingType(change.Old[0].Type) {
						return c.deleteURLForwardingRecord(dc.Name, id)
					}
					return c.deleteRecord(dc.Name, id)
				},
			}
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *porkbunProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := c.getRecords(domain)
	if err != nil {
		return nil, err
	}
	forwards, err := c.getURLForwardingRecords(domain)
	if err != nil {
		return nil, err
	}
	existingRecords := make(models.Records, 0)
	for i := range records {
		shouldSkip := false
		if strings.HasSuffix(records[i].Content, ".porkbun.com") {
			name := dc.ToShort(records[i].Name)
			if name == "@" {
				name = ""
			}
			if records[i].Type == "ALIAS" {
				for _, forward := range forwards {
					if name == forward.Subdomain {
						shouldSkip = true
						break
					}
				}
			}
			if records[i].Type == "CNAME" {
				for _, forward := range forwards {
					if name == "*."+forward.Subdomain {
						shouldSkip = true
						break
					}
				}
			}
		}
		if shouldSkip {
			continue
		}
		newr, err := toRc(dc, &records[i])
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, newr)
	}
	for i := range forwards {
		r := &forwards[i]
		recordType := "URL"
		if r.Type == "permanent" {
			recordType = "URL301"
		}
		rc, err := dc.NewRecordConfig(dc.LabelFromShort(r.Subdomain), 0, recordType, r.Location)
		if err != nil {
			return nil, err
		}
		rc.Original = r
		rc.Metadata = map[string]string{
			metaType:        r.Type,
			metaIncludePath: r.IncludePath,
			metaWildcard:    r.Wildcard,
		}
		existingRecords = append(existingRecords, rc)
	}
	return existingRecords, nil
}

// parses the porkbun format into our standard RecordConfig.
func toRc(dc *models.DomainConfig, r *domainRecord) (*models.RecordConfig, error) {
	ttlValue, _ := strconv.ParseUint(r.TTL, 10, 32)
	ttl := uint32(ttlValue)
	priority, _ := strconv.ParseInt(r.Prio, 10, 16)
	label := dc.LabelFromFQDNNoDot(r.Name)

	var rc *models.RecordConfig
	var err error
	switch rtype := r.Type; rtype { // #rtype_variations
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, r.Content)
	case "MX":
		target := r.Content
		if !strings.HasSuffix(target, ".") {
			target += "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, priority, target)
	case "CNAME", "ALIAS", "NS":
		target := r.Content
		if !strings.HasSuffix(target, ".") {
			target += "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, rtype, target)
	case "CAA":
		// 0, issue, "letsencrypt.org"
		c := strings.Split(r.Content, " ")

		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeCAA, c[0], c[1], strings.ReplaceAll(c[2], "\"", ""))
	case "TLSA":
		// 0 0 0 00000000000000000000000
		c := strings.Split(r.Content, " ")

		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTLSA, c[0], c[1], c[2], c[3])
	case "SRV":
		// 5 5060 sip.example.com
		c := strings.Split(r.Content, " ")

		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSRV, priority, c[0], c[1], c[2])
	case "HTTPS":
		fallthrough
	case "SVCB":
		// 5 . ech=AAAAABBBBB...
		c := strings.Split(r.Content, " ")

		params := ""
		if len(c) > 2 {
			params = strings.Join(c[2:], " ")
		}
		rc, err = dc.NewRecordConfig(label, ttl, rtype, c[0], c[1], params)
	case "SSHFP":
		rc, err = dc.NewRecordConfigParse(label, ttl, dnsv2.TypeSSHFP, r.Content)
	default:
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Content)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	return rc, err
}

// toReq takes a RecordConfig and turns it into the native format used by the API.
func toReq(rc *models.RecordConfig) (requestParams, error) {
	if isURLForwardingType(rc.Type) {
		subdomain := rc.GetLabel()
		if subdomain == "@" {
			subdomain = ""
		}
		return requestParams{
			"subdomain":   subdomain,
			"location":    rc.GetTargetField(),
			"type":        rc.Metadata[metaType],
			"includePath": rc.Metadata[metaIncludePath],
			"wildcard":    rc.Metadata[metaWildcard],
		}, nil
	}

	req := requestParams{
		"type":    rc.Type,
		"name":    rc.GetLabel(),
		"content": rc.GetTargetField(),
		"ttl":     strconv.Itoa(int(rc.TTL)),
	}

	// porkbun doesn't use "@", it uses an empty name
	if req["name"] == "@" {
		req["name"] = ""
	}

	switch rc.Type { // #rtype_variations
	case "A", "AAAA", "NS", "ALIAS", "CNAME":
	// Nothing special.
	case "TXT":
		req["content"] = rc.GetTargetTXTJoined()
	case "MX":
		req["prio"] = strconv.Itoa(int(rc.MxPreference))
	case "SRV":
		req["prio"] = strconv.Itoa(int(rc.SrvPriority))
		req["content"] = fmt.Sprintf("%d %d %s", rc.SrvWeight, rc.SrvPort, rc.GetTargetField())
	case "CAA":
		req["content"] = fmt.Sprintf("%d %s \"%s\"", rc.CaaFlag, rc.CaaTag, rc.GetTargetField())
	case "TLSA":
		req["content"] = fmt.Sprintf("%d %d %d %s",
			rc.TlsaUsage, rc.TlsaSelector, rc.TlsaMatchingType, rc.GetTargetField())
	case "HTTPS":
		fallthrough
	case "SVCB":
		f := rc.AsSVCB()
		req["content"] = fmt.Sprintf("%d %s %s",
			rc.SvcPriority, rc.GetTargetField(), models.Svcbv2ValueToString(f.Value))
	case "SSHFP":
		req["content"] = fmt.Sprintf("%v %v %s",
			rc.SshfpAlgorithm, rc.SshfpFingerprint, rc.GetTargetField())
	default:
		return nil, fmt.Errorf("porkbun.toReq rtype %q unimplemented", rc.Type)
	}

	return req, nil
}

func checkNSModifications(dc *models.DomainConfig) {
	newList := make(models.Records, 0, len(dc.Records))
	for _, rec := range dc.Records {
		if rec.Type == "NS" && rec.GetLabelFQDN() == dc.Name {
			if strings.HasSuffix(rec.GetTargetField(), ".porkbun.com") {
				printer.Warnf("porkbun does not support modifying NS records on base domain. %s will not be added.\n", rec.GetTargetField())
			}
			continue
		}
		newList = append(newList, rec)
	}
	dc.Records = newList
}

func fixTTL(ttl uint32) uint32 {
	if ttl > minimumTTL {
		return ttl
	}
	return minimumTTL
}

func (c *porkbunProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	nss, err := c.getNameservers(dc.Name)
	if err != nil {
		return nil, err
	}
	foundNameservers := strings.Join(nss, ",")

	expected := make([]string, 0, len(dc.Nameservers))
	for _, ns := range dc.Nameservers {
		expected = append(expected, strings.ToLower(strings.TrimRight(ns.Name, ".")))
	}
	slices.Sort(expected)
	expected = slices.Compact(expected)
	expectedNameservers := strings.Join(expected, ",")

	if foundNameservers == expectedNameservers {
		return nil, nil
	}

	return []*models.Correction{
		{
			Msg: fmt.Sprintf("Update nameservers %s -> %s", foundNameservers, expectedNameservers),
			F: func() error {
				return c.updateNameservers(expected, dc.Name)
			},
		},
	}, nil
}
