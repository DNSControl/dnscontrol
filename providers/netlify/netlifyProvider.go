package netlify

import (
	"encoding/json"
	"errors"
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
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
	observer    providers.ConversionObserver
	apiToken    string // the account access token
	accountSlug string // the account identifier slug. optional.
}

func (n *netlifyProvider) SetConversionObserver(observer providers.ConversionObserver) {
	n.observer = observer
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
		before := providers.BeginToRC(n.observer, "toRecordConfig", r)
		rec, err := toRecordConfig(dc, r)
		providers.EndToRC(n.observer, "toRecordConfig", before, r, models.Records{rec}, err)
		if err != nil {
			return nil, err
		}
		if rec == nil {
			continue
		}

		cleanRecords = append(cleanRecords, rec)
	}

	return cleanRecords, nil
}

// toRecordConfig converts a Netlify record to a RecordConfig. It returns nil for
// SOA records and for the NETLIFY and NETLIFYv6 pseudo-types, which are ignored.
func toRecordConfig(dc *models.DomainConfig, r *dnsRecord) (*models.RecordConfig, error) {
	if r.Type == "SOA" {
		return nil, nil
	}

	label := dc.LabelFromFQDNNoDot(r.Hostname) // Netlify returns the FQDN.
	ttl := uint32(r.TTL)

	var rec *models.RecordConfig
	var err error
	switch rtype := r.Type; rtype {
	case "NETLIFY", "NETLIFYv6": // transparently ignore
		return nil, nil
	case "MX":
		rec, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, r.Priority, r.Value,
			nrc.Flags{TargetIsFqdnNoDot: true})
	case "SRV":
		rec, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSRV, r.Priority, r.Weight, r.Port, r.Value,
			nrc.Flags{TargetIsFqdnNoDot: true})
	case "TXT":
		rec, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, r.Value)
	case "CAA":
		rec, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeCAA, r.Flag, r.Tag, r.Value,
			nrc.Flags{TargetIsFqdnNoDot: true})
	default:
		rec, err = dc.NewRecordConfigParse(label, ttl, r.Type, r.Value,
			nrc.Flags{TargetIsFqdnNoDot: true})
	}

	if err != nil {
		return nil, fmt.Errorf("unparsable record received from Netlify: %w", err)
	}

	rec.Original = r

	return rec, nil
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
	changes, actualChangeCount, err := diff2.ByRecord(records, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	zone, err := n.getZone(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})

		case diff2.CREATE:
			input := models.Records{change.New[0]}
			before := providers.BeginToNative(n.observer, "toReq", input)
			req := toReq(change.New[0])
			providers.EndToNative(n.observer, "toReq", before, input, req, nil)
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					_, err := n.createDNSRecord(zone.ID, req)
					return err
				},
			})

		case diff2.DELETE:
			id := change.Old[0].Original.(*dnsRecord).ID
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return n.deleteDNSRecord(zone.ID, id)
				},
			})

		case diff2.CHANGE:
			// Netlify has no update API, so a change is a delete followed by a create.
			id := change.Old[0].Original.(*dnsRecord).ID
			input := models.Records{change.New[0]}
			before := providers.BeginToNative(n.observer, "toReq", input)
			req := toReq(change.New[0])
			providers.EndToNative(n.observer, "toReq", before, input, req, nil)
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					if err := n.deleteDNSRecord(zone.ID, id); err != nil {
						return err
					}

					_, err := n.createDNSRecord(zone.ID, req)
					return err
				},
			})

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

func toReq(rc *models.RecordConfig) *dnsRecordCreate {
	name := rc.GetLabelFQDN() // Netlify wants the FQDN

	r := &dnsRecordCreate{
		Type:     rc.Type,
		Hostname: name,
		TTL:      int64(rc.TTL),
	}

	switch f := rc.GetRDATA().(type) {
	case dnsrdatav2.CAA:
		r.Tag = f.Tag
		r.Flag = int64(f.Flag)
		r.Value = f.Value
	case dnsrdatav2.MX:
		r.Priority = int64(f.Preference)
		r.Value = f.Mx
	case dnsrdatav2.SRV:
		r.Priority = int64(f.Priority)
		r.Port = int64(f.Port)
		r.Weight = int64(f.Weight)
		r.Value = f.Target
	case dnsrdatav2.TXT:
		r.Value = rc.GetTargetTXTJoined()
	default:
		r.Value = rc.GetRDATA().String()
	}

	return r
}
