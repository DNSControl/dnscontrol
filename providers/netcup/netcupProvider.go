package netcup

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
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
	existingRecords := make([]*models.RecordConfig, len(records))
	for i := range records {
		existingRecords[i] = toRecordConfig(domain, &records[i])
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

	toReport, create, del, modify, actualChangeCount, err := diff.NewCompat(dc).IncrementalDiff(existingRecords)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	var recordsToUpdate []record
	var correctionMsgs []string

	// Collect all deletions
	for _, d := range del {
		req := d.Existing.Original.(*record)
		req.Delete = true // Mark for deletion
		recordsToUpdate = append(recordsToUpdate, *req)
		correctionMsgs = append(correctionMsgs, fmt.Sprintf("%s, Netcup ID: %s", d.String(), req.ID))
	}

	// Collect all creations
	for _, c := range create {
		req := fromRecordConfig(c.Desired)
		req.Delete = false // Mark for creation
		recordsToUpdate = append(recordsToUpdate, *req)
		correctionMsgs = append(correctionMsgs, c.String())
	}

	// Collect all modifications
	for _, m := range modify {
		req := fromRecordConfig(m.Desired)
		req.ID = m.Existing.Original.(*record).ID // Preserve original ID for modification
		req.Delete = false                        // Mark for modification
		recordsToUpdate = append(recordsToUpdate, *req)
		correctionMsgs = append(correctionMsgs, fmt.Sprintf("%s, Netcup ID: %s", m.String(), req.ID))
	}

	if len(recordsToUpdate) > 0 {
		corr := &models.Correction{
			Msg: strings.Join(correctionMsgs, "\n"),
			F: func() error {
				data := paramUpdateRecords{
					Key:            api.credentials.apikey,
					SessionID:      api.credentials.sessionID,
					CustomerNumber: api.credentials.customernumber,
					DomainName:     domain,
					RecordSet:      records{Records: recordsToUpdate},
				}
				_, err := api.get("updateDnsRecords", data)
				if err != nil {
					return fmt.Errorf("error while trying to update records in bulk: %w", err)
				}
				return nil
			},
		}
		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}
