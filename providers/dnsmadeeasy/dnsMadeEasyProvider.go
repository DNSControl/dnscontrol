package dnsmadeeasy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func init() {
	providers.Register[*dnsMadeEasyProvider]("DNSMADEEASY", providers.Definition{
		FriendlyName: "DNS Made Easy",
		PortalURL:    "https://cp.dnsmadeeasy.com/",
		CredFields: []providers.CredsField{
			{
				Key:      "api_key",
				Label:    "API key",
				Help:     "Your DNS Made Easy API key.",
				Secret:   true,
				Required: true,
			},
			{
				Key:      "secret_key",
				Label:    "Secret key",
				Help:     "Your DNS Made Easy secret key.",
				Secret:   true,
				Required: true,
			},
			{
				Key:          "sandbox",
				Label:        "Use the DNS Made Easy sandbox API instead of production?",
				Help:         "Answer no for normal use. The sandbox needs its own API credentials from https://sandbox.dnsmadeeasy.com/.",
				ConfirmValue: "1",
			},
		},
		Maintainer: "@vojtad",
		SupportedTypes: []string{
			"Default",
			"ALIAS",
			"IMPORT_TRANSFORM",
			"PTR",
		},
		CanConcur:              providers.Unimplemented(),
		CanUseDSForChildren:    providers.Cannot(),
		DocDualHost:            providers.Can("System NS records cannot be edited. Custom apex NS records can be added/changed/deleted."),
		DocOfficiallySupported: providers.Cannot(),
	})
}

// Initialize initializes a fresh provider instance.
func (api *dnsMadeEasyProvider) Initialize(settings map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	if settings["api_key"] == "" {
		return errors.New("missing DNSMADEEASY api_key")
	}

	if settings["secret_key"] == "" {
		return errors.New("missing DNSMADEEASY secret_key")
	}

	sandbox := settings["sandbox"] != ""

	debug := os.Getenv("DNSMADEEASY_DEBUG_HTTP") == "1"

	api.initializeClient(settings["api_key"], settings["secret_key"], sandbox, debug)

	return nil
}

func (api *dnsMadeEasyProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	domainName := dc.Name
	domainID, err := api.findDomainID(domainName)
	if err != nil {
		return nil, 0, err
	}

	for _, rec := range dc.Records {
		// NS records have a fixed TTL on DNS Made Easy that cannot be changed.
		if rec.Type == "NS" {
			rec.TTL = fixedNameServerRecordTTL
		}
	}

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction

	// DNS Made Easy applies creates, deletes and modifications in three
	// separate batch API calls, so collect each category before emitting.
	var deleteRecordIds []int
	deleteDescription := []string{"Batch deletion of records:"}
	var createRecords []recordRequestData
	createDescription := []string{"Batch creation of records:"}
	var modifyRecords []recordRequestData
	modifyDescription := []string{"Batch modification of records:"}

	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})

		case diff2.DELETE:
			originalRecordID := change.Old[0].Original.(*recordResponseDataEntry).ID
			deleteRecordIds = append(deleteRecordIds, originalRecordID)
			deleteDescription = append(deleteDescription, change.Msgs[0])

		case diff2.CREATE:
			record := fromRecordConfig(change.New[0])
			createRecords = append(createRecords, *record)
			createDescription = append(createDescription, change.Msgs[0])

		case diff2.CHANGE:
			originalRecord := change.Old[0].Original.(*recordResponseDataEntry)

			record := fromRecordConfig(change.New[0])
			record.ID = originalRecord.ID
			record.GtdLocation = originalRecord.GtdLocation

			modifyRecords = append(modifyRecords, *record)
			modifyDescription = append(modifyDescription, change.Msgs[0])

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	if len(deleteRecordIds) > 0 {
		corr := &models.Correction{
			Msg: strings.Join(deleteDescription, "\n\t"),
			F: func() error {
				return api.deleteRecords(domainID, deleteRecordIds)
			},
		}
		corrections = append(corrections, corr)
	}

	if len(createRecords) > 0 {
		corr := &models.Correction{
			Msg: strings.Join(createDescription, "\n\t"),
			F: func() error {
				return api.createRecords(domainID, createRecords)
			},
		}
		corrections = append(corrections, corr)
	}

	if len(modifyRecords) > 0 {
		corr := &models.Correction{
			Msg: strings.Join(modifyDescription, "\n\t"),
			F: func() error {
				return api.updateRecords(domainID, modifyRecords)
			},
		}
		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (api *dnsMadeEasyProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name

	exists, err := api.domainExists(domain)
	if err != nil {
		return err
	}

	// domain already exists
	if exists {
		return nil
	}

	return api.createDomain(domain)
}

// GetNameservers returns the nameservers for a domain.
func (api *dnsMadeEasyProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	nameServers, err := api.fetchDomainNameServers(domain)
	if err != nil {
		return nil, err
	}

	return models.ToNameservers(nameServers)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *dnsMadeEasyProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := api.fetchDomainRecords(domain)
	if err != nil {
		return nil, err
	}

	nameServers, err := api.fetchDomainNameServers(domain)
	if err != nil {
		return nil, err
	}

	existingRecords := make(models.Records, 0, len(records))
	for i := range records {
		// Ignore HTTPRED and SPF records
		if records[i].Type == "HTTPRED" || records[i].Type == "SPF" {
			continue
		}
		existingRecords = append(existingRecords, toRecordConfig(dc, &records[i]))
	}

	for i := range nameServers {
		existingRecords = append(existingRecords, systemNameServerToRecordConfig(dc, nameServers[i]))
	}

	return existingRecords, nil
}

// ListZones lists the zones on this account.
func (api *dnsMadeEasyProvider) ListZones() ([]string, error) {
	if err := api.loadDomains(); err != nil {
		return nil, err
	}

	var zones []string
	for i := range api.domains {
		zones = append(zones, i)
	}

	return zones, nil
}
