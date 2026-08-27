package packetframe

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// packetframeProvider is the handle for this provider.
type packetframeProvider struct {
	observer    providers.ConversionObserver
	client      *http.Client
	baseURL     *url.URL
	token       string
	domainIndex map[string]zoneInfo
}

func (api *packetframeProvider) SetConversionObserver(observer providers.ConversionObserver) {
	api.observer = observer
}

// newPacketframe creates the provider.
func newPacketframe(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	if m["token"] == "" {
		return nil, errors.New("missing Packetframe token")
	}

	baseURL, err := url.Parse(defaultBaseURL)
	if err != nil {
		return nil, errors.New("invalid base URL for Packetframe")
	}
	client := http.Client{}

	api := &packetframeProvider{client: &client, baseURL: baseURL, token: m["token"]}

	return api, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Unimplemented(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "PACKETFRAME"
	const providerMaintainer = "NEEDS VOLUNTEER"
	fns := providers.DspFuncs{
		Initializer:   newPacketframe,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

// GetNameservers returns the nameservers for a domain.
func (api *packetframeProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(defaultNameServerNames)
}

func (api *packetframeProvider) getZone(domain string) (*zoneInfo, error) {
	if api.domainIndex == nil {
		if err := api.fetchDomainList(); err != nil {
			return nil, err
		}
	}
	z, ok := api.domainIndex[domain+"."]
	if !ok {
		return nil, fmt.Errorf("%q not a zone in Packetframe account", domain)
	}

	return &z, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *packetframeProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	zone, err := api.getZone(domain)
	if err != nil {
		return nil, fmt.Errorf("no such zone %q in Packetframe account", domain)
	}

	records, err := api.getRecords(zone.ID)
	if err != nil {
		return nil, fmt.Errorf("could not load records for domain %q", domain)
	}

	existingRecords := make(models.Records, len(records))

	for i := range records {
		before := providers.BeginToRC(api.observer, "toRc", &records[i])
		existingRecords[i], err = toRc(dc, &records[i])
		providers.EndToRC(api.observer, "toRc", before, &records[i], models.Records{existingRecords[i]}, err)
		if err != nil {
			return nil, err
		}
	}

	return existingRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *packetframeProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	zone, err := api.getZone(dc.Name)
	if err != nil {
		return nil, 0, fmt.Errorf("no such zone %q in Packetframe account", dc.Name)
	}

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})

		case diff2.CREATE:
			before := providers.BeginToNative(api.observer, "toReq", change.New)
			req, err := toReq(zone.ID, change.New[0])
			providers.EndToNative(api.observer, "toReq", before, change.New, req, err)
			if err != nil {
				return nil, 0, err
			}
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					_, err := api.createRecord(req)
					return err
				},
			})

		case diff2.DELETE:
			original := change.Old[0].Original.(*domainRecord)
			if original.ID == "0" { // Skip the default nameservers
				continue
			}
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return api.deleteRecord(zone.ID, original.ID)
				},
			})

		case diff2.CHANGE:
			original := change.Old[0].Original.(*domainRecord)
			if original.ID == "0" { // Skip the default nameservers
				continue
			}
			before := providers.BeginToNative(api.observer, "toReq", change.New)
			req, err := toReq(zone.ID, change.New[0])
			providers.EndToNative(api.observer, "toReq", before, change.New, req, err)
			if err != nil {
				return nil, 0, err
			}
			req.ID = original.ID
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return api.modifyRecord(req)
				},
			})

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

func toReq(zoneID string, rc *models.RecordConfig) (*domainRecord, error) {
	req := &domainRecord{
		Type:  rc.Type,
		TTL:   int(rc.TTL),
		Label: rc.GetLabel(),
		Zone:  zoneID,
	}

	switch rc.TypeNum {
	case dnsv2.TypeTXT:
		req.Value = rc.GetTargetTXTJoined()
	default:
		req.Value = rc.GetRDATA().String()
	}

	return req, nil
}

func toRc(dc *models.DomainConfig, r *domainRecord) (*models.RecordConfig, error) {
	label := strings.TrimSuffix(r.Label, dc.Name+".")
	label = strings.TrimSuffix(label, ".")
	label = dc.LabelFromShort(label)
	ttl := uint32(r.TTL)

	var rc *models.RecordConfig
	var err error
	switch rtype := r.Type; rtype {
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Value)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, r.Value)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	return rc, err
}
