package netcup

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
	// We make an API call to verify that we have authority for this domain.
	if _, err := api.getRecords(domain); err != nil {
		return nil, err
	}

	return models.ToNameservers([]string{
		"root-dns.netcup.net",
		"second-dns.netcup.net",
		"third-dns.netcup.net",
	})
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *netcupProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	domain := dc.Name

	// Create a copy of the desired state for diffing.
	// This is to avoid side-effects on the original DomainConfig.
	dcForDiff := *dc
	// We filter out unsupported record types (NS) and zero out the TTL
	// as it is not supported by the provider.
	desiredRecords := make(models.Records, 0, len(dc.Records))
	for _, r := range dc.Records {
		if r.Type == "NS" {
			continue
		}
		rec := *r
		rec.TTL = 0
		desiredRecords = append(desiredRecords, &rec)
	}
	dcForDiff.Records = desiredRecords

	instructions, actualChangeCount, err := diff2.ByRecord(existingRecords, &dcForDiff, nil)
	if err != nil {
		return nil, 0, err
	}

	recordsToUpdate := []record{}
	msgs := []string{}
	var corrections []*models.Correction

	for _, inst := range instructions {
		switch inst.Type {
		case diff2.REPORT:
			// Handle report-only corrections separately. They don't involve API calls.
			corrections = append(corrections, &models.Correction{Msg: inst.MsgsJoined})
			continue
		case diff2.DELETE:
			native := inst.Old[0].Original.(*record)
			// For deletion, we must send the original record and set the delete flag.
			// The API validates other fields even on deletion, so we make a copy and modify it.
			recToDelete := *native
			recToDelete.Delete = true
			recordsToUpdate = append(recordsToUpdate, recToDelete)
			msgs = append(msgs, inst.MsgsJoined)
		case diff2.CREATE:
			rec := fromRecordConfig(inst.New[0])
			recordsToUpdate = append(recordsToUpdate, *rec)
			msgs = append(msgs, inst.MsgsJoined)
		case diff2.CHANGE:
			native := inst.Old[0].Original.(*record)
			rec := fromRecordConfig(inst.New[0])
			rec.ID = native.ID
			recordsToUpdate = append(recordsToUpdate, *rec)
			msgs = append(msgs, inst.MsgsJoined)
		}
	}

	if len(recordsToUpdate) > 0 {
		// Create one big correction for the batch update.
		batchCorrection := &models.Correction{
			Msg: strings.Join(msgs, "\n"),
			F: func() error {
				return api.updateRecords(domain, recordsToUpdate)
			},
		}
		corrections = append(corrections, batchCorrection)
	}

	return corrections, actualChangeCount, nil
}

func (api *netcupProvider) updateRecords(domain string, recs []record) error {
	payload := paramUpdateRecords{
		Key:            api.credentials.apikey,
		SessionID:      api.credentials.sessionID,
		CustomerNumber: api.credentials.customernumber,
		DomainName:     domain,
		RecordSet:      records{Records: recs},
	}
	_, err := api.get("updateDnsRecords", payload)
	if err != nil {
		return fmt.Errorf("failed while trying to update records (netcup): %w", err)
	}
	return nil
}
