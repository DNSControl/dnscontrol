package hetzner

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/DNSControl/dnscontrol/v4/pkg/zonecache"
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
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "HETZNER"
	const providerMaintainer = "@das7pad"
	fns := providers.DspFuncs{
		Initializer:   New,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

// New creates a new API handle.
func New(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	apiKey := settings["api_key"]
	if apiKey == "" {
		return nil, errors.New("missing HETZNER api_key")
	}

	api := &hetznerProvider{apiKey: apiKey}
	api.zoneCache = zonecache.New(api.fetchAllZones)
	return api, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (api *hetznerProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	if ok, err := api.zoneCache.HasZone(domain); err != nil || ok {
		return err
	}

	if err := api.createZone(domain); err != nil {
		return err
	}
	return nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *hetznerProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	domain := dc.Name

	toReport, create, del, modify, actualChangeCount, err := diff.NewCompat(dc).IncrementalDiff(existingRecords)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	z, err := api.zoneCache.GetZone(domain)
	if err != nil {
		return nil, 0, err
	}

	for _, m := range del {
		r := m.Existing.Original.(*record)
		corr := &models.Correction{
			Msg: m.String(),
			F: func() error {
				return api.deleteRecord(r)
			},
		}
		corrections = append(corrections, corr)
	}

	createRecords := make([]record, len(create))
	createDescription := make([]string, len(create)+1)
	createDescription[0] = "Batch creation of records:"
	for i, m := range create {
		createRecords[i] = fromRecordConfig(m.Desired, z)
		createDescription[i+1] = m.String()
	}
	if len(createRecords) > 0 {
		corr := &models.Correction{
			Msg: strings.Join(createDescription, "\n\t"),
			F: func() error {
				return api.bulkCreateRecords(createRecords)
			},
		}
		corrections = append(corrections, corr)
	}

	modifyRecords := make([]record, len(modify))
	modifyDescription := make([]string, len(modify)+1)
	modifyDescription[0] = "Batch modification of records:"
	for i, m := range modify {
		r := fromRecordConfig(m.Desired, z)
		r.ID = m.Existing.Original.(*record).ID
		modifyRecords[i] = r
		modifyDescription[i+1] = m.String()
	}
	if len(modifyRecords) > 0 {
		corr := &models.Correction{
			Msg: strings.Join(modifyDescription, "\n\t"),
			F: func() error {
				return api.bulkUpdateRecords(modifyRecords)
			},
		}
		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

// GetNameservers returns the nameservers for a domain.
func (api *hetznerProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	z, err := api.zoneCache.GetZone(domain)
	if err != nil {
		return nil, err
	}

	return models.ToNameserversStripTD(z.NameServers)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *hetznerProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := api.getAllRecords(domain)
	if err != nil {
		return nil, err
	}
	existingRecords := make([]*models.RecordConfig, len(records))
	for i := range records {
		existingRecords[i], err = toRecordConfig(domain, &records[i])
		if err != nil {
			return nil, err
		}
	}
	return existingRecords, nil
}

// ListZones lists the zones on this account.
func (api *hetznerProvider) ListZones() ([]string, error) {
	return api.zoneCache.GetZoneNames()
}
