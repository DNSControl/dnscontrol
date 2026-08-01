package netcup

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Cannot(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Cannot(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "NETCUP"
	const providerMaintainer = "@kordianbruck"
	fns := providers.DspFuncs{
		Initializer:   New,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

// New creates a new API handle.
func New(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	if settings["api-key"] == "" || settings["api-password"] == "" || settings["customer-number"] == "" {
		return nil, errors.New("missing netcup login parameters")
	}

	api := &netcupProvider{}
	err := api.login(settings["api-key"], settings["api-password"], settings["customer-number"])
	if err != nil {
		return nil, fmt.Errorf("login to netcup DNS failed, please check your credentials: %w", err)
	}
	return api, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *netcupProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := api.getRecords(domain)
	if err != nil {
		return nil, err
	}
	existingRecords := make(models.Records, len(records))
	for i := range records {
		existingRecords[i], err = toRecordConfig(dc, &records[i])
		if err != nil {
			return nil, err
		}
	}

	return existingRecords, nil
}

// GetNameservers returns the nameservers for a domain.
// As netcup doesn't support setting nameservers over this API, these are static.
// Domains not managed by netcup DNS will return an error.
func (api *netcupProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers([]string{
		"root-dns.netcup.net",
		"second-dns.netcup.net",
		"third-dns.netcup.net",
	})
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *netcupProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	domain := dc.Name

	// Setting the TTL is not supported for netcup
	for _, r := range dc.Records {
		r.TTL = 0
	}

	// Filter out types we can't modify (like NS)
	newRecords := models.Records{}
	for _, r := range dc.Records {
		if r.Type != "NS" {
			newRecords = append(newRecords, r)
		}
	}
	dc.Records = newRecords

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
			req := fromRecordConfig(change.New[0])
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return api.createRecord(domain, req)
				},
			})

		case diff2.DELETE:
			req := change.Old[0].Original.(*record)
			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("%s, Netcup ID: %s", change.Msgs[0], req.ID),
				F: func() error {
					return api.deleteRecord(domain, req)
				},
			})

		case diff2.CHANGE:
			id := change.Old[0].Original.(*record).ID
			req := fromRecordConfig(change.New[0])
			req.ID = id
			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("%s, Netcup ID: %s: ", change.Msgs[0], id),
				F: func() error {
					return api.modifyRecord(domain, req)
				},
			})

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	return corrections, actualChangeCount, nil
}
