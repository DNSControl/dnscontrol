package hetznerv2

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"golang.org/x/net/idna"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/DNSControl/dnscontrol/v5/pkg/version"
	"github.com/DNSControl/dnscontrol/v5/pkg/zonecache"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDS:               providers.Can(),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
	providers.DocDualHost:            providers.Can(),
}

func init() {
	const providerName = "HETZNER_V2"
	const providerMaintainer = "@das7pad"
	fns := providers.DspFuncs{
		Initializer:   New,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Hetzner DNS",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/hetzner_v2",
		PortalURL:   "https://console.hetzner.com/projects",
		Fields: []providers.CredsField{
			{
				Key:      "api_token",
				Label:    "API token",
				Help:     "Your Hetzner Cloud API token.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// New creates a new API handle.
func New(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	apiToken := settings["api_token"]
	if apiToken == "" {
		return nil, errors.New("missing HETZNER_V2 api_token")
	}

	h := &hetznerv2Provider{
		client: hcloud.NewClient(
			hcloud.WithToken(apiToken),
			hcloud.WithApplication("dnscontrol", version.Version()),
		),
	}
	h.zoneCache = zonecache.New(h.fetchAllZones)
	return h, nil
}

type hetznerv2Provider struct {
	zoneCache zonecache.ZoneCache[*hcloud.Zone]
	client    *hcloud.Client
}

// fetchAllZones is used by the zonecache.ZoneCache.
func (h *hetznerv2Provider) fetchAllZones() (map[string]*hcloud.Zone, error) {
	flat, err := h.client.Zone.All(context.Background())
	if err != nil {
		return nil, err
	}
	zones := make(map[string]*hcloud.Zone, len(flat))
	for _, z := range flat {
		zones[z.Name] = z
	}
	return zones, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (h *hetznerv2Provider) EnsureZoneExists(dc *models.DomainConfig) error {
	if ok, err2 := h.zoneCache.HasZone(dc.Name); err2 != nil || ok {
		return err2
	}
	result, _, err := h.client.Zone.Create(context.Background(), hcloud.ZoneCreateOpts{
		Name: dc.Name,
		Mode: hcloud.ZoneModePrimary,
	})
	if err != nil {
		return err
	}
	err = h.client.Action.WaitFor(context.Background(), result.Action)
	if err != nil {
		return err
	}
	z, _, err := h.client.Zone.GetByID(context.Background(), result.Zone.ID)
	if err != nil {
		return err
	}
	h.zoneCache.SetZone(dc.Name, z)
	return nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (h *hetznerv2Provider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	z, err := h.zoneCache.GetZone(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	// Hetzner Cloud has a "ByRecordSet" API for DNS.
	// At each label:rtype pair, we either delete all records or UPSERT the desired records.
	instructions, actualChangeCount, err := diff2.ByRecordSet(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var reports []*models.Correction
	for _, instruction := range instructions {
		switch instruction.Type {
		case diff2.REPORT:
			reports = append(reports, &models.Correction{
				Msg: instruction.MsgsJoined,
			})
			continue
		case diff2.CREATE:
			first := instruction.New[0]
			ttl := int(first.TTL)
			opts := hcloud.ZoneRRSetCreateOpts{
				Name: first.Name,
				Type: hcloud.ZoneRRSetType(first.Type),
				TTL:  &ttl,
			}
			for _, r := range instruction.New {
				opts.Records = append(opts.Records, hcloud.ZoneRRSetRecord{
					Value: r.GetRDATA().String(),
				})
			}
			reports = append(reports, &models.Correction{
				F: func() error {
					_, _, err2 := h.client.Zone.CreateRRSet(context.Background(), z, opts)
					return err2
				},
				Msg: instruction.MsgsJoined,
			})
		case diff2.CHANGE:
			rrSet := instruction.Old[0].Original.(*hcloud.ZoneRRSet)
			reports = append(reports, &models.Correction{
				F: func() error {
					if instruction.New[0].TTL != instruction.Old[0].TTL {
						ttl := int(instruction.New[0].TTL)
						opts := hcloud.ZoneRRSetChangeTTLOpts{TTL: &ttl}
						_, _, err2 := h.client.Zone.ChangeRRSetTTL(context.Background(), rrSet, opts)
						if err2 != nil {
							return err2
						}
					}

					opts := hcloud.ZoneRRSetSetRecordsOpts{}
					for _, r := range instruction.New {
						opts.Records = append(opts.Records, hcloud.ZoneRRSetRecord{
							Value: r.GetRDATA().String(),
						})
					}
					_, _, err2 := h.client.Zone.SetRRSetRecords(context.Background(), rrSet, opts)
					return err2
				},
				Msg: instruction.MsgsJoined,
			})
		case diff2.DELETE:
			reports = append(reports, &models.Correction{
				F: func() error {
					rc := instruction.Old[0].Original.(*hcloud.ZoneRRSet)
					_, _, err2 := h.client.Zone.DeleteRRSet(context.Background(), rc)
					return err2
				},
				Msg: instruction.MsgsJoined,
			})
		}
	}

	return reports, actualChangeCount, nil
}

// GetNameservers returns the nameservers for a domain.
func (h *hetznerv2Provider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	encoded, err := idna.ToASCII(domain)
	if err != nil {
		return nil, err
	}
	z, err := h.zoneCache.GetZone(encoded)
	if err != nil {
		return nil, err
	}
	return models.ToNameserversStripTD(z.AuthoritativeNameservers.Assigned)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (h *hetznerv2Provider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	z, err := h.zoneCache.GetZone(dc.Name)
	if err != nil {
		return nil, err
	}
	opts := hcloud.ZoneRRSetListOpts{
		PerPage: 100}
	records, err := h.client.Zone.AllRRSetsWithOpts(context.Background(), z, opts)
	if err != nil {
		return nil, err
	}
	existingRecords := make(models.Records, 0, len(records))
	for _, rrSet := range records {
		recs, err := nativeToRecords(dc, rrSet, uint32(z.TTL))
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, recs...)
	}
	return existingRecords, nil
}

// nativeToRecords converts a Hetzner RRSet to RecordConfigs, one per value.
// zoneTTL is the TTL of the zone the RRSet belongs to, used when the RRSet does
// not carry one of its own. It returns nothing for SOA RRSets, which are hidden.
func nativeToRecords(dc *models.DomainConfig, rrSet *hcloud.ZoneRRSet, zoneTTL uint32) (models.Records, error) {
	if rrSet.Type == hcloud.ZoneRRSetTypeSOA {
		// SOA records are not available for editing, hide them.
		return nil, nil
	}
	ttl := zoneTTL
	if rrSet.TTL != nil {
		ttl = uint32(*rrSet.TTL)
	}

	recs := make(models.Records, 0, len(rrSet.Records))
	for _, r := range rrSet.Records {
		rc, err := dc.NewRecordConfigParse(dc.LabelFromShort(rrSet.Name), ttl, string(rrSet.Type), r.Value)
		if err != nil {
			return nil, err
		}
		rc.Original = rrSet
		recs = append(recs, rc)
	}
	return recs, nil
}

// ListZones lists the zones on this account.
func (h *hetznerv2Provider) ListZones() ([]string, error) {
	return h.zoneCache.GetZoneNames()
}
