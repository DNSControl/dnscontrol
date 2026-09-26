package mittwald

import (
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	mwdns "github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/dnsv2"
)

/*
mStudio keeps one "DNS zone" per name: the domain itself and every name below
it (www, _dmarc, _autodiscover._tcp) are separate zones. A zone has one slot
per record set (A and AAAA together, CNAME, MX, TXT, SRV, CAA); a slot is
replaced as a whole or unset. Record sets that mStudio manages (the addresses
of an ingress, the mail exchangers of mittwald's mail service) are neither
returned nor changed.
*/

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	// The API knows A, AAAA, CNAME, MX, TXT, SRV and CAA only.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanConcur:              providers.Cannot(),
	providers.CanGetZones:            providers.Cannot(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseDNAME:            providers.Cannot(),
	providers.CanUseDNSKEY:           providers.Cannot(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseHTTPS:            providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSMIMEA:           providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseSVCB:             providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot("A domain's zone exists once the domain is in an mStudio project"),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "MITTWALD"
	const providerMaintainer = "@twiesing"
	fns := providers.DspFuncs{
		Initializer:   newProvider,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "mittwald mStudio",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/mittwald",
		PortalURL:   "https://studio.mittwald.de",
		Fields: []providers.CredsField{
			{Key: "api_token", Label: "API token", Help: "An mStudio API token of a user with access to the projects of the domains.", Required: true, Secret: true},
		},
	})
}

type mittwaldProvider struct {
	api      *api
	observer providers.ConversionObserver

	mu    sync.Mutex
	zones map[string]map[string]mwdns.Zone // domain -> name -> zone, as last read by GetZoneRecords
}

func newProvider(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	token := settings["api_token"]
	if token == "" {
		return nil, errors.New("missing MITTWALD api_token")
	}
	a, err := newAPI(token)
	if err != nil {
		return nil, err
	}
	return &mittwaldProvider{api: a, zones: map[string]map[string]mwdns.Zone{}}, nil
}

// SetConversionObserver lets the integration tests record conversions for the golden files.
func (p *mittwaldProvider) SetConversionObserver(observer providers.ConversionObserver) {
	p.observer = observer
}

// GetNameservers returns nothing: mStudio does not let NS records be managed.
func (p *mittwaldProvider) GetNameservers(string) ([]*models.Nameserver, error) {
	return nil, nil
}

// GetZoneRecords returns the records of the custom record sets of the domain and every name below it.
func (p *mittwaldProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	zones, err := p.api.zonesOf(dc.Name)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	p.zones[dc.Name] = zones
	p.mu.Unlock()

	var recs models.Records
	for _, name := range slices.Sorted(maps.Keys(zones)) {
		z := zones[name]
		before := providers.BeginToRC(p.observer, "toRC", &z)
		r, err := toRC(dc, z)
		providers.EndToRC(p.observer, "toRC", before, &z, r, err)
		if err != nil {
			return nil, err
		}
		recs = append(recs, r...)
	}
	return recs, nil
}

// GetZoneRecordsCorrections returns the changes that turn the existing records into dc.Records.
func (p *mittwaldProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	p.mu.Lock()
	zones := p.zones[dc.Name]
	p.mu.Unlock()
	root, ok := zones[dc.Name]
	if !ok {
		return nil, 0, errors.New("GetZoneRecords must run before GetZoneRecordsCorrections")
	}

	prepDesiredRecords(dc)

	// A zone holds every record set of one name, so a name is the unit of change.
	instructions, count, err := diff2.ByLabel(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, inst := range instructions {
		name := inst.Key.NameFQDN
		zone, exists := zones[name]
		old, err := slotsOf(inst.Old)
		if err != nil {
			return nil, 0, err
		}

		switch inst.Type {
		case diff2.REPORT:
			corrections = append(corrections, inst.CreateMessage())

		case diff2.CREATE, diff2.CHANGE:
			before := providers.BeginToNative(p.observer, "toNative", inst.New)
			bodies, err := toNative(inst.New)
			providers.EndToNative(p.observer, "toNative", before, inst.New, bodies, err)
			if err != nil {
				return nil, 0, err
			}
			label := strings.TrimSuffix(name, "."+dc.Name)
			corrections = append(corrections, inst.CreateCorrection(func() error {
				zoneID := zone.Id
				if !exists {
					id, err := p.api.createZone(root.Id, label)
					if err != nil {
						return err
					}
					zoneID = id
				}
				// Unset first: a CNAME can only be set on a name without other records.
				for _, set := range old {
					if _, keep := bodies[set]; !keep {
						if err := p.api.unsetRecordSet(zoneID, set); err != nil {
							return err
						}
					}
				}
				for _, set := range slices.Sorted(maps.Keys(bodies)) {
					if err := p.api.setRecordSet(zoneID, set, bodies[set]); err != nil {
						return err
					}
				}
				return nil
			}))

		case diff2.DELETE:
			keep := keepZone(dc.Name, name, zone, zones)
			corrections = append(corrections, inst.CreateCorrection(func() error {
				if !keep {
					// Once a zone is deleted while it has a CAA set, mStudio
					// answers every CAA set of a new zone of that name with 500;
					// a set unset before does not do this.
					if slices.Contains(old, setCAA) {
						if err := p.api.unsetRecordSet(zone.Id, setCAA); err != nil {
							return err
						}
					}
					return p.api.deleteZone(zone.Id)
				}
				for _, set := range old {
					if err := p.api.unsetRecordSet(zone.Id, set); err != nil {
						return err
					}
				}
				return nil
			}))
		}
	}
	return corrections, count, nil
}

// keepZone reports whether the zone of name stays when its records go: the
// domain's own zone does, a zone with a set that mStudio manages, and a zone
// with other zones below it.
func keepZone(domain, name string, zone mwdns.Zone, zones map[string]mwdns.Zone) bool {
	if name == domain || hasManagedSet(zone) {
		return true
	}
	for other := range zones {
		if strings.HasSuffix(other, "."+name) {
			return true
		}
	}
	return false
}

// slotsOf returns the record sets that hold the records, each once, sorted.
func slotsOf(recs models.Records) ([]domainclientv2.UpdateRecordSetRequestPathRecordSet, error) {
	sets := map[domainclientv2.UpdateRecordSetRequestPathRecordSet]bool{}
	for _, rc := range recs {
		set, err := slotOf(rc.TypeNum)
		if err != nil {
			return nil, err
		}
		sets[set] = true
	}
	return slices.Sorted(maps.Keys(sets)), nil
}
