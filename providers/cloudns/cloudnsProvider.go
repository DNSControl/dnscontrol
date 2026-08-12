package cloudns

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/fatih/color"
)

/*
ClouDNS API DNS provider:
Info required in `creds.json`:
   - auth-id or sub-auth-id
   - auth-password
*/

func newCloudns(m map[string]string) (*cloudnsProvider, error) {
	c := &cloudnsProvider{}
	c.requestLimit = NewAdaptiveLimiter(10, 10)

	c.creds.id, c.creds.password, c.creds.subid = m["auth-id"], m["auth-password"], m["sub-auth-id"]

	if (c.creds.id == "" && c.creds.subid == "") || c.creds.password == "" {
		return nil, errors.New("missing ClouDNS auth-id or sub-auth-id and auth-password")
	}

	return c, nil
}

func newDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newCloudns(conf)
}

func newReg(conf map[string]string) (providers.Registrar, error) {
	return newCloudns(conf)
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDNAME:            providers.Can(),
	providers.CanUseDSForChildren:    providers.Can(),
	providers.CanUseLOC:              providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUseSOA:              providers.Unimplemented("Supported by cloudns at a separate API endpoint (/dns/modify-soa.json), not implemented yet"),
	providers.CanUseDS:               providers.Cannot("Not supported for root, only for children"),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseSVCB:             providers.Cannot(),
	providers.CanUseHTTPS:            providers.Cannot(),
	providers.CanUseDNSKEY:           providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "CLOUDNS"
	const providerMaintainer = "@pragmaton"
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterRegistrarType(providerName, newReg)
	providers.RegisterCustomRecordType("CLOUDNS_WR", providerName, "")
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "ClouDNS",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/cloudns",
		PortalURL:   "https://www.cloudns.net/api-settings/",
		Notes:       "ClouDNS supports two auth methods: a main API user (auth-id) or a sub-user API account (sub-auth-id). Both use the same auth-password.",
		Fields: []providers.CredsField{
			{
				Key:      "_authMethod",
				Label:    "Which authentication method do you want to use?",
				Help:     "Choose whether to authenticate with your main API auth-id or with a sub-user sub-auth-id.",
				Choices:  []string{"auth-id", "sub-auth-id"},
				Required: true,
				Internal: true,
			},
			{
				Key:      "auth-id",
				Label:    "Auth ID",
				Help:     "Your ClouDNS API auth-id.",
				Required: true,
				ShowIf:   map[string]string{"_authMethod": "auth-id"},
			},
			{
				Key:      "sub-auth-id",
				Label:    "Sub-auth ID",
				Help:     "Your ClouDNS sub-user API sub-auth-id.",
				Required: true,
				ShowIf:   map[string]string{"_authMethod": "sub-auth-id"},
			},
			{
				Key:      "auth-password",
				Label:    "Auth password",
				Help:     "The API password associated with the chosen auth-id or sub-auth-id.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// GetNameservers returns the nameservers for a domain.
func (c *cloudnsProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	names, err := c.fetchAvailableNameservers()
	if err != nil {
		return nil, err
	}

	return models.ToNameservers(names)
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *cloudnsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	domainID, ok, err := c.idForDomain(dc.Name)
	if err != nil {
		return nil, 0, err
	} else if !ok {
		return nil, 0, fmt.Errorf("'%s' not a zone in ClouDNS account", dc.Name)
	}

	// Get a list of available TTL values.
	// The TTL list needs to be obtained for each domain, so get it first here.
	allowedTTLValues, err := c.fetchAvailableTTLValues(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	// ClouDNS can only be specified from a specific TTL list, so change the TTL in advance.
	for _, record := range dc.Records {
		record.TTL = fixTTL(allowedTTLValues, record.TTL)
	}

	dnssecFixes, err := c.getDNSSECCorrections(dc)
	if err != nil {
		return nil, 0, err
	}

	instructions, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, compareMetadata)
	if err != nil {
		return nil, 0, err
	}

	var (
		reportMsgs []string
		create     diff.Changeset
		del        diff.Changeset
		modify     diff.Changeset
	)
	for _, inst := range instructions {
		cor := diff.Correlation{}
		switch inst.Type {
		case diff2.REPORT:
			reportMsgs = append(reportMsgs, inst.Msgs...)
		case diff2.CREATE:
			cor.Desired = inst.New[0]
			create = append(create, cor)
		case diff2.CHANGE:
			cor.Existing = inst.Old[0]
			cor.Desired = inst.New[0]
			modify = append(modify, cor)
		case diff2.DELETE:
			cor.Existing = inst.Old[0]
			del = append(del, cor)
		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(reportMsgs)
	corrections = append(corrections, dnssecFixes...)

	// Deletes first so changing type works etc.
	for _, m := range del {
		id := m.Existing.Original.(*domainRecord).ID
		corr := &models.Correction{
			Msg: fmt.Sprintf("%s%s, ClouDNS ID: %s", m.String(), addMetadataCorrection(m.Existing, m.Desired), id),
			F: func() error {
				return c.deleteRecord(domainID, id)
			},
		}
		// at ClouDNS, we MUST have a NS for a DS
		// So, when deleting, we must delete the DS first, otherwise deleting the NS throws an error
		if m.Existing.Type == "DS" {
			// type DS is prepended - so executed first
			corrections = append([]*models.Correction{corr}, corrections...)
		} else {
			corrections = append(corrections, corr)
		}
	}

	var (
		createCorrections         []*models.Correction
		createARecordCorrections  []*models.Correction
		createNSRecordCorrections []*models.Correction
	)
	for _, m := range create {
		input := models.Records{m.Desired}
		before := providers.BeginToNative(c.observer, "toReq", input)
		req, err := toReq(m.Desired)
		providers.EndToNative(c.observer, "toReq", before, input, req, err)
		if err != nil {
			return nil, 0, err
		}

		// ClouDNS does not require the trailing period to be specified when creating an NS record where the A or AAAA record exists in the zone.
		// So, modify it to remove the trailing period.
		if req["record-type"] == "NS" && strings.HasSuffix(req["record"], domainID+".") {
			req["record"] = strings.TrimSuffix(req["record"], ".")
		}

		corr := &models.Correction{
			Msg: fmt.Sprintf("%s%s", m.String(), addMetadataCorrection(m.Existing, m.Desired)),
			F: func() error {
				return c.createRecord(domainID, req)
			},
		}
		// A & AAAA need to be created before NS #2244
		// NS need to be created before DS #1018
		// or else errors will be thrown
		switch m.Desired.TypeNum {
		case dnsv2.TypeA, dnsv2.TypeAAAA:
			createARecordCorrections = append(createARecordCorrections, corr)
		case dnsv2.TypeNS:
			createNSRecordCorrections = append(createNSRecordCorrections, corr)
		default:
			createCorrections = append(createCorrections, corr)
		}
	}
	corrections = append(corrections, createARecordCorrections...)
	corrections = append(corrections, createNSRecordCorrections...)
	corrections = append(corrections, createCorrections...)

	for _, m := range modify {
		id := m.Existing.Original.(*domainRecord).ID
		input := models.Records{m.Desired}
		before := providers.BeginToNative(c.observer, "toReq", input)
		req, err := toReq(m.Desired)
		providers.EndToNative(c.observer, "toReq", before, input, req, err)
		if err != nil {
			return nil, 0, err
		}

		// ClouDNS does not require the trailing period to be specified when updating an NS record where the A or AAAA record exists in the zone.
		// So, modify it to remove the trailing period.
		if req["record-type"] == "NS" && strings.HasSuffix(req["record"], domainID+".") {
			req["record"] = strings.TrimSuffix(req["record"], ".")
		}

		corr := &models.Correction{
			Msg: fmt.Sprintf("%s%s, ClouDNS ID: %s: ", m.String(), addMetadataCorrection(m.Existing, m.Desired), id),
			F: func() error {
				return c.modifyRecord(domainID, id, req)
			},
		}
		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

// GetRegistrarCorrections returns corrections to update domain nameserver delegation.
func (c *cloudnsProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	// 1. Get current nameservers from registrar
	existing, err := c.getNameservers(dc.Name)
	if err != nil {
		return nil, err
	}
	sort.Strings(existing)
	existingStr := strings.Join(existing, ",")

	// 2. Get desired nameservers from config
	desired := models.NameserversToStrings(dc.Nameservers)
	sort.Strings(desired)
	desiredStr := strings.Join(desired, ",")

	// 3. Compare and return correction if needed
	if existingStr != desiredStr {
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Update nameservers from %q to %q", existingStr, desiredStr),
				F: func() error {
					return c.setNameservers(dc.Name, desired)
				},
			},
		}, nil
	}

	return nil, nil
}

// getDNSSECCorrections returns corrections that update a domain's DNSSEC state.
func (c *cloudnsProvider) getDNSSECCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	enabled, err := c.isDnssecEnabled(dc.Name)
	if err != nil {
		return nil, err
	}

	if enabled && dc.AutoDNSSEC == "off" {
		return []*models.Correction{
			{
				Msg: "Disable DNSSEC",
				F:   func() error { err := c.setDnssec(dc.Name, false); return err },
			},
		}, nil
	}

	if !enabled && dc.AutoDNSSEC == "on" {
		return []*models.Correction{
			{
				Msg: "Enable DNSSEC",
				F:   func() error { err := c.setDnssec(dc.Name, true); return err },
			},
		}, nil
	}

	return []*models.Correction{}, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *cloudnsProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	records, err := c.getRecords(dc.Name)
	if err != nil {
		return nil, err
	}
	existingRecords := make([]*models.RecordConfig, len(records))
	for i := range records {
		before := providers.BeginToRC(c.observer, "toRc", &records[i])
		existingRecords[i], err = toRc(dc, &records[i])
		providers.EndToRC(c.observer, "toRc", before, &records[i], models.Records{existingRecords[i]}, err)
		if err != nil {
			return nil, err
		}
	}
	return existingRecords, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (c *cloudnsProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	if _, ok, err := c.idForDomain(domain); err != nil {
		return err
	} else if ok { // zone already exists
		return nil
	}
	return c.createDomain(domain)
}

// ListZones returns names of all DNS zones managed by this provider.
func (c *cloudnsProvider) ListZones() ([]string, error) {
	if err := c.fetchZones(); err != nil {
		return nil, err
	}

	zones := make([]string, 0, len(c.domainIndex))
	for zone := range c.domainIndex {
		zones = append(zones, zone)
	}

	return zones, nil
}

// parses the ClouDNS format into our standard RecordConfig.
func toRc(dc *models.DomainConfig, r *domainRecord) (*models.RecordConfig, error) {
	var err error

	ttl_, err := strconv.ParseUint(r.TTL, 10, 32)
	if err != nil {
		return nil, err
	}
	ttl := uint32(ttl_)
	label := dc.LabelFromShort(r.Host)

	var rc *models.RecordConfig
	switch rtype := r.Type; rtype { // #rtype_variations
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, r.Target)
	case "MX":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, r.Priority, dc.ToFqdnWithDot(r.Target+".")) // ignore:legacyfield
	case "SRV":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSRV, r.Priority, r.Weight, r.Port, dc.ToFqdnWithDot(r.Target+".")) // ignore:legacyfield
	case "ALIAS":
		rc, err = dc.NewRecordConfig(label, ttl, privatetypes.TypeALIAS, dc.ToFqdnWithDot(r.Target+"."))
	case "CNAME", "DNAME", "NS", "PTR":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, dc.ToFqdnWithDot(r.Target+"."))
	case "CAA":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeCAA, r.CaaFlag, r.CaaTag, r.CaaValue) // ignore:legacyfield
	case "TLSA":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTLSA, r.TlsaUsage, r.TlsaSelector, r.TlsaMatchingType, r.Target) // ignore:legacyfield
	case "SSHFP":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSSHFP, r.SshfpAlgorithm, r.SshfpFingerprint, r.Target) // ignore:legacyfield
	case "DS":
		// SshfpAlgorithm and DS algorithm both use the API's "algorithm" field.
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeDS, r.DsKeyTag, r.SshfpAlgorithm, r.DsDigestType, r.Target) // ignore:legacyfield
	case "CLOUD_WR":
		rc, err = dc.NewRecordConfig(label, ttl, privatetypes.TypeCLOUDNSWR, r.Target)
	case "LOC":
		latSec, err := parseFloat32(r.LocLatSec)
		if err != nil {
			return nil, err
		}

		longSec, err := parseFloat32(r.LocLongSec)
		if err != nil {
			return nil, err
		}

		altitude, err := parseFloat32(r.LocAltitude) // ignore:legacyfield
		if err != nil {
			return nil, err
		}

		size, err := parseFloat32(r.LocSize) // ignore:legacyfield
		if err != nil {
			return nil, err
		}

		hPrec, err := parseFloat32(r.LocHPrecision)
		if err != nil {
			return nil, err
		}

		vPrec, err := parseFloat32(r.LocVPrecision)
		if err != nil {
			return nil, err
		}

		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeLOC,
			r.LocLatDeg, r.LocLatMin, latSec, r.LocLatDir,
			r.LocLongDeg, r.LocLongMin, longSec, r.LocLongDir,
			altitude, size, hPrec, vPrec)
		if err != nil {
			return nil, err
		}

	case "NAPTR":
		target := dc.ToFqdnWithDot(r.NaptrReplacement + ".")
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeNAPTR, r.NaptrOrder, r.NaptrPreference, r.NaptrFlags, r.NaptrService, r.NaptrRegexp, target) // ignore:legacyfield
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, r.Target)
	}
	if err != nil {
		return nil, err
	}

	// Add metadata for GeoDNS
	// Note: By default, it works only with A, AAAA, CNAME, NAPTR or SRV record
	// but you can ask the support for others type of record and they enable it
	// for your ClouDNS account.
	if r.GeodnsCode != "" {
		rc.Metadata[metaGeodnsCode] = r.GeodnsCode
	}

	rc.Original = r

	// Add metadata for GeoDNS. By default, it works only with A, AAAA, CNAME,
	// NAPTR, or SRV records, but support can enable other types per account.
	if r.GeodnsCode != "" {
		rc.Metadata[metaGeodnsCode] = r.GeodnsCode
	}

	return rc, nil
}

// parseFloat32 parses s into a float32. This has an advantage over
// NewRecordConfig because errors are detected earlier.
func parseFloat32(s string) (float32, error) {
	f, err := strconv.ParseFloat(s, 32)
	return float32(f), err
}

func formatLocParam(param string) string {
	param = strings.Split(param, "m")[0]
	// API misbehaves with a parameter of "0.00" and treats it as the default, so convert to "0" for this case only
	if param == "0.00" {
		param = "0"
	}
	return param
}

// toReq takes a RecordConfig and turns it into the native format used by the API.
func toReq(rc *models.RecordConfig) (requestParams, error) {

	host := rc.GetLabel()
	// ClouDNS doesn't use "@", it uses an empty name
	if host == "@" {
		host = ""
	}

	req := requestParams{
		"record-type": rc.Type,
		"host":        host,
		"ttl":         strconv.Itoa(int(rc.TTL)),
	}

	// Add metadata for GeoDNS
	// Note: By default, it works only with A, AAAA, CNAME, NAPTR or SRV record
	// but you can ask the support for others type of record and they enable it
	// for your ClouDNS account.
	if geodnsCodeFromMetadataValue, ok := rc.Metadata[metaGeodnsCode]; ok {
		req["geodns-code"] = geodnsCodeFromMetadataValue
	}

	switch rc.TypeNum {
	case privatetypes.TypeCLOUDNSWR:
		f := rc.AsCLOUDNSWR()
		req["record-type"] = "WR"
		req["record"] = f.Target
	case dnsv2.TypeMX:
		f := rc.AsMX()
		req["priority"] = strconv.Itoa(int(f.Preference))
		req["record"] = f.Mx
	case dnsv2.TypeSRV:
		f := rc.AsSRV()
		req["priority"] = strconv.Itoa(int(f.Priority))
		req["weight"] = strconv.Itoa(int(f.Weight))
		req["port"] = strconv.Itoa(int(f.Port))
		req["record"] = f.Target
	case dnsv2.TypeCAA:
		f := rc.AsCAA()
		req["caa_flag"] = strconv.Itoa(int(f.Flag))
		req["caa_type"] = f.Tag
		req["caa_value"] = f.Value
		req["record"] = rc.GetRDATA().String()
	case dnsv2.TypeTLSA:
		f := rc.AsTLSA()
		req["tlsa_usage"] = strconv.Itoa(int(f.Usage))
		req["tlsa_selector"] = strconv.Itoa(int(f.Selector))
		req["tlsa_matching_type"] = strconv.Itoa(int(f.MatchingType))
		req["record"] = f.Certificate
	case dnsv2.TypeSSHFP:
		f := rc.AsSSHFP()
		req["algorithm"] = strconv.Itoa(int(f.Algorithm))
		req["fptype"] = strconv.Itoa(int(f.Type))
		req["record"] = f.FingerPrint
	case dnsv2.TypeDS:
		f := rc.AsDS()
		req["key-tag"] = strconv.Itoa(int(f.KeyTag))
		req["algorithm"] = strconv.Itoa(int(f.Algorithm))
		req["digest-type"] = strconv.Itoa(int(f.DigestType))
		req["record"] = f.Digest
	case dnsv2.TypeLOC:
		parts := strings.Fields(rc.GetRDATA().String())
		req["lat-deg"] = parts[0]
		req["lat-min"] = parts[1]
		req["lat-sec"] = parts[2]
		req["lat-dir"] = parts[3]
		req["long-deg"] = parts[4]
		req["long-min"] = parts[5]
		req["long-sec"] = parts[6]
		req["long-dir"] = parts[7]
		req["altitude"] = formatLocParam(parts[8])
		req["size"] = formatLocParam(parts[9])
		req["h-precision"] = formatLocParam(parts[10])
		req["v-precision"] = formatLocParam(parts[11])
		req["record"] = rc.GetRDATA().String()
	case dnsv2.TypeNAPTR:
		f := rc.AsNAPTR()
		req["order"] = strconv.Itoa(int(f.Order))
		req["pref"] = strconv.Itoa(int(f.Preference))
		req["flag"] = f.Flags
		req["params"] = f.Service
		req["regexp"] = f.Regexp
		req["replace"] = f.Replacement
		req["record"] = rc.GetRDATA().String()
	case dnsv2.TypeTXT:
		req["record"] = rc.GetTargetTXTJoined()
	default:
		req["record"] = rc.GetRDATA().String()
	}

	return req, nil
}

func addMetadataCorrection(existingRc *models.RecordConfig, desiredRc *models.RecordConfig) string {
	if existingRc == nil {
		if desiredRc.Metadata != nil {
			geodnsCodeFromMetadataValue, geodnsCodeFromMetadataExist := desiredRc.Metadata[metaGeodnsCode]
			if geodnsCodeFromMetadataExist {
				return color.GreenString(fmt.Sprintf(" location=%s", geodnsCodeFromMetadataValue))
			}
		}

		return ""
	}
	if desiredRc == nil {
		if existingRc.Metadata != nil {
			geodnsCodeFromMetadataValue, geodnsCodeFromMetadataExist := existingRc.Metadata[metaGeodnsCode]
			if geodnsCodeFromMetadataExist {
				return color.RedString(fmt.Sprintf(" location=%s", geodnsCodeFromMetadataValue))
			}
		}

		return ""
	}

	if compareMetadata(existingRc) == compareMetadata(desiredRc) {
		return ""
	}

	// By default, the value is "DEFAULT"
	// To compare geodns metadata, we replace the value "DEFAULT" with an empty string
	// Here, we replace the empty string with "DEFAULT", so the end user can see the
	// real value send to the provider.

	geodnsCodeFromExistingRcMetadataValue, geodnsCodeFromExistingRcMetadataExist := existingRc.Metadata[metaGeodnsCode]
	geodnsCodeFromDesiredRcMetadataValue, geodnsCodeFromDesiredRcMetadataExist := desiredRc.Metadata[metaGeodnsCode]

	if !geodnsCodeFromExistingRcMetadataExist {
		geodnsCodeFromExistingRcMetadataValue = "DEFAULT"
	}

	if !geodnsCodeFromDesiredRcMetadataExist {
		geodnsCodeFromDesiredRcMetadataValue = "DEFAULT"
	}

	return color.YellowString(", (location=%s) -> (location=%s)", geodnsCodeFromExistingRcMetadataValue, geodnsCodeFromDesiredRcMetadataValue)
}

func compareMetadata(rc *models.RecordConfig) string {
	if len(rc.Metadata) == 0 {
		return ""
	}

	// To compare GeoDNS metadata, we must temporary remove the value "DEFAULT".
	// - DNS record without GeoDNS return ""
	// - DNS record with GeoDNS return "DEFAULT" as empty value
	val, exist := rc.Metadata[metaGeodnsCode]
	if exist && val == "DEFAULT" {
		delete(rc.Metadata, metaGeodnsCode)
	}

	// If there is no metadata left, we return an empty string (like the first if)
	// Useful for the case where a DNS record has the "DEFAULT" GeoDNS value
	// or the account doesn't have GeoDNS feature enabled
	if len(rc.Metadata) == 0 {
		return ""
	}

	// json.Marshal always serialize all fields in alphabetical order,
	// so the metadata string will be consistent between runs
	result, err := json.Marshal(rc.Metadata)
	if err != nil {
		printer.Warnf("Cannot serialize metadata of record %s", rc)
		return ""
	}

	// Restore the metadata value
	if exist && val == "DEFAULT" {
		rc.Metadata[metaGeodnsCode] = "DEFAULT"
	}

	return string(result)
}
