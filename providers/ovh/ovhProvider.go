package ovh

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
	"github.com/ovh/go-ovh/ovh"
)

type ovhProvider struct {
	client *ovh.Client
	zones  map[string]bool
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseLOC:              providers.Unimplemented(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Cannot("New domains require registration"),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func newOVH(m map[string]string, _ json.RawMessage) (*ovhProvider, error) {
	appKey, appSecretKey, consumerKey := m["app-key"], m["app-secret-key"], m["consumer-key"]

	c, err := ovh.NewClient(getOVHEndpoint(m), appKey, appSecretKey, consumerKey)
	if c == nil {
		return nil, err
	}

	ovh := &ovhProvider{client: c}
	if err := ovh.fetchZones(); err != nil {
		return nil, err
	}
	return ovh, nil
}

func getOVHEndpoint(params map[string]string) string {
	if ep, ok := params["endpoint"]; ok && ep != "" {
		switch strings.ToLower(ep) {
		case "eu":
			return ovh.OvhEU
		case "ca":
			return ovh.OvhCA
		case "us":
			return ovh.OvhUS
		default:
			return ep
		}
	}
	return ovh.OvhEU
}

func newDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newOVH(conf, metadata)
}

func newReg(conf map[string]string) (providers.Registrar, error) {
	return newOVH(conf, nil)
}

func init() {
	const providerName = "OVH"
	const providerMaintainer = "@masterzen"
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterRegistrarType(providerName, newReg)
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

func (c *ovhProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	_, ok := c.zones[domain]
	if !ok {
		return nil, fmt.Errorf("'%s' not a zone in ovh account", domain)
	}

	ns, err := c.fetchNS(domain)
	if err != nil {
		return nil, err
	}

	return models.ToNameservers(ns)
}

type errNoExist struct {
	domain string
}

func (e errNoExist) Error() string {
	return fmt.Sprintf("Domain %s not found in your ovh account", e.domain)
}

// ListZones lists the zones on this account.
func (c *ovhProvider) ListZones() (zones []string, err error) {
	for zone := range c.zones {
		zones = append(zones, zone)
	}
	return
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *ovhProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	if !c.zones[domain] {
		return nil, errNoExist{domain}
	}

	records, err := c.fetchRecords(domain)
	if err != nil {
		return nil, err
	}

	var actual models.Records
	for _, r := range records {
		rec, err := nativeToRecord(r, dc)
		if err != nil {
			return nil, err
		}
		actual = append(actual, rec)
	}

	return actual, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *ovhProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, actual models.Records) ([]*models.Correction, int, error) {
	corrections, actualChangeCount, err := c.getDiff2DomainCorrections(dc, actual)
	if err != nil {
		return nil, 0, err
	}

	// Only refresh zone if there's a real modification
	reportOnlyCorrections := true
	for _, c := range corrections {
		if c.F != nil {
			reportOnlyCorrections = false
			break
		}
	}

	if !reportOnlyCorrections {
		corrections = append(corrections, &models.Correction{
			Msg: "REFRESH zone " + dc.Name,
			F: func() error {
				return c.refreshZone(dc.Name)
			},
		})
	}

	return corrections, actualChangeCount, nil
}

func (c *ovhProvider) getDiff2DomainCorrections(dc *models.DomainConfig, actual models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction
	instructions, actualChangeCount, err := diff2.ByRecord(actual, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	for _, inst := range instructions {
		switch inst.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: inst.MsgsJoined})
		case diff2.CHANGE:
			corrections = append(corrections, &models.Correction{
				Msg: inst.Msgs[0],
				F:   c.updateRecordFunc(inst.Old[0].Original.(*Record), inst.New[0], dc.Name),
			})
		case diff2.CREATE:
			corrections = append(corrections, &models.Correction{
				Msg: inst.Msgs[0],
				F:   c.createRecordFunc(inst.New[0], dc.Name),
			})
		case diff2.DELETE:
			rec := inst.Old[0].Original.(*Record)
			corrections = append(corrections, &models.Correction{
				Msg: inst.Msgs[0],
				F:   c.deleteRecordFunc(rec.ID, dc.Name),
			})
		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}
	return corrections, actualChangeCount, nil
}

func nativeToRecord(r *Record, dc *models.DomainConfig) (*models.RecordConfig, error) {

	if r.FieldType == "SOA" {
		return nil, nil
	}

	// ovh uses a custom type for SPF, DKIM and DMARC.
	rtype := r.FieldType
	if rtype == "SPF" || rtype == "DKIM" || rtype == "DMARC" {
		rtype = "TXT"
	}

	label := dc.LabelFromShort(r.SubDomain)

	// ovh default is 3600
	ttl := uint32(r.TTL)
	if ttl == 0 {
		ttl = 3600
	}

	var rc *models.RecordConfig
	var err error

	switch rtype {
	case "TXT":
		var tx string
		switch r.FieldType {
		case "DKIM", "DMARC":
			// Unlike regular TXT and SPF records, OVH stores and returns DKIM
			// and DMARC targets as raw, unquoted text (see adaptNativeRecord,
			// which strips quotes before writing them). Running raw text
			// through the RFC1035 zone-file TXT parser is wrong: an unescaped
			// ';' is interpreted as the start of a comment, silently
			// truncating values like "v=DMARC1; p=none; rua=...".
			tx = r.Target
		default:
			tx, err = txtutil.ParseQuoted(r.Target)
			if err != nil {
				return nil, err
			}
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, tx)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, r.Target)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r

	return rc, nil
}

func (c *ovhProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	// get the actual in-use nameservers
	actualNs, err := c.fetchRegistrarNS(dc.Name)
	if err != nil {
		return nil, err
	}

	// get the actual used ones + the configured one through dnscontrol
	expectedNs := []string{}
	for _, d := range dc.Nameservers {
		expectedNs = append(expectedNs, d.Name)
	}

	sort.Strings(actualNs)
	actual := strings.Join(actualNs, ",")

	sort.Strings(expectedNs)
	expected := strings.Join(expectedNs, ",")

	// check if we need to change something
	if actual != expected {
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Change Nameservers from '%s' to '%s'", actual, expected),
				F: func() error {
					err := c.updateNS(dc.Name, expectedNs)
					if err != nil {
						return err
					}
					return nil
				},
			},
		}, nil
	}

	return nil, nil
}
