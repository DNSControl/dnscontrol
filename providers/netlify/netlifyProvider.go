package netlify

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot(),
	providers.DocDualHost:            providers.Cannot("Netlify does not allow sufficient control over the apex NS records"),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "NETLIFY"
	const providerMaintainer = "@SphericalKat"
	fns := providers.DspFuncs{
		Initializer:   newNetlify,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterCustomRecordType(providerName, providerName, "")
	providers.RegisterCustomRecordType("NETLIFYv6", providerName, "")
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

type netlifyProvider struct {
	apiToken    string // the account access token
	accountSlug string // the account identifier slug. optional.
}

func newNetlify(m map[string]string, message json.RawMessage) (providers.DNSServiceProvider, error) {
	api := &netlifyProvider{}
	api.apiToken = m["token"]
	if api.apiToken == "" {
		return nil, errors.New("missing Netlify personal access token")
	}

	api.accountSlug = m["slug"]

	return api, nil
}

func (n *netlifyProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	zone, err := n.getZone(domain)
	if err != nil {
		return nil, err
	}

	return models.ToNameservers(zone.DNSServers)
}

func (n *netlifyProvider) getZone(domain string) (*dnsZone, error) {
	zones, err := n.getDNSZones()
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		if zone.Name == domain {
			return zone, nil
		}
	}

	return nil, errors.New("no zones found for this domain")
}

func (n *netlifyProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	zone, err := n.getZone(domain)
	if err != nil {
		return nil, err
	}

	records, err := n.getDNSRecords(zone.ID)
	if err != nil {
		return nil, err
	}

	cleanRecords := make(models.Records, 0)

	for _, r := range records {
		if r.Type == "SOA" {
			continue
		}

		rec := &models.RecordConfig{
			TTL:      uint32(r.TTL),
			Original: r,
		}

		rec.SetLabelFromFQDN(r.Hostname, domain) // netlify returns the FQDN

		if r.Type == "CNAME" || r.Type == "MX" || r.Type == "NS" {
			r.Value = dnsutil.Canonical(r.Value)
		}

		switch rtype := r.Type; rtype {
		case "NETLIFY", "NETLIFYv6": // transparently ignore
			continue
		case "MX":
			err = rec.SetTargetMX(uint16(r.Priority), r.Value)
		case "SRV":
			parts := strings.Fields(r.Value)
			if len(parts) == 3 {
				r.Value += "."
			}
			err = rec.SetTargetSRV(uint16(r.Priority), r.Weight, r.Port, r.Value)
		case "TXT":
			err = rec.SetTargetTXT(r.Value)
		case "CAA":
			err = rec.SetTargetCAA(uint8(r.Flag), r.Tag, r.Value)
		default:
			err = rec.PopulateFromString(r.Type, r.Value, domain)
		}

		if err != nil {
			return nil, fmt.Errorf("unparsable record received from Netlify: %w", err)
		}

		cleanRecords = append(cleanRecords, rec)
	}

	return cleanRecords, nil
}

// ListZones returns all DNS zones managed by this provider.
func (n *netlifyProvider) ListZones() ([]string, error) {
	zones, err := n.getDNSZones()
	if err != nil {
		return nil, err
	}

	zoneNames := make([]string, len(zones))
	for i, z := range zones {
		zoneNames[i] = z.Name
	}

	return zoneNames, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (n *netlifyProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, records models.Records) ([]*models.Correction, int, error) {
	toReport, create, del, modify, actualChangeCount, err := diff.NewCompat(dc).IncrementalDiff(records)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	zone, err := n.getZone(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	// Deletes first so changing type works etc.
	for _, m := range del {
		id := m.Existing.Original.(*dnsRecord).ID
		corr := &models.Correction{
			Msg: m.String(),
			F: func() error {
				return n.deleteDNSRecord(zone.ID, id)
			},
		}
		corrections = append(corrections, corr)
	}

	for _, m := range create {
		req := toReq(m.Desired)
		corr := &models.Correction{
			Msg: m.String(),
			F: func() error {
				_, err := n.createDNSRecord(zone.ID, req)
				return err
			},
		}
		corrections = append(corrections, corr)
	}

	for _, m := range modify {
		id := m.Existing.Original.(*dnsRecord).ID
		req := toReq(m.Desired)
		corr := &models.Correction{
			Msg: m.String(),
			F: func() error {
				if err := n.deleteDNSRecord(zone.ID, id); err != nil {
					return err
				}

				_, err := n.createDNSRecord(zone.ID, req)
				return err
			},
		}
		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

func toReq(rc *models.RecordConfig) *dnsRecordCreate {
	name := rc.GetLabelFQDN() // Netlify wants the FQDN
	target := rc.GetTargetField()
	priority := int64(0)

	switch rc.Type {
	case "MX":
		priority = int64(rc.MxPreference)
	case "SRV":
		priority = int64(rc.SrvPriority)
	case "TXT":
		target = rc.GetTargetTXTJoined()
	default:
		// no action required
	}

	return &dnsRecordCreate{
		Type:     rc.Type,
		Hostname: name,
		Value:    target,
		TTL:      int64(rc.TTL),
		Priority: priority,
		Port:     int64(rc.SrvPort),
		Weight:   int64(rc.SrvWeight),
		Tag:      rc.CaaTag,
		Flag:     int64(rc.CaaFlag),
	}
}
