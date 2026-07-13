package dnsmadeeasy

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can("System NS records cannot be edited. Custom apex NS records can be added/changed/deleted."),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "DNSMADEEASY"
	const providerMaintainer = "@vojtad"
	fns := providers.DspFuncs{
		Initializer:   New,
		RecordAuditor: AuditRecords,
	}

	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "DNS Made Easy",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/dnsmadeeasy",
		PortalURL:   "https://cp.dnsmadeeasy.com/",
		Fields: []providers.CredsField{
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
				Key:   "sandbox",
				Label: "Use sandbox (optional)",
				Help:  "Set to any non-empty value to use the DNS Made Easy sandbox API instead of production.",
			},
		},
	})
}

// New creates a new API handle.
func New(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	if settings["api_key"] == "" {
		return nil, errors.New("missing DNSMADEEASY api_key")
	}

	if settings["secret_key"] == "" {
		return nil, errors.New("missing DNSMADEEASY secret_key")
	}

	sandbox := settings["sandbox"] != ""

	debug := os.Getenv("DNSMADEEASY_DEBUG_HTTP") == "1"

	api := newProvider(settings["api_key"], settings["secret_key"], sandbox, debug)

	return api, nil
}

func (api *dnsMadeEasyProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	domainName := dc.Name
	domainID, err := api.findDomainID(domainName)
	if err != nil {
		return nil, 0, err
	}

	for _, rec := range dc.Records {
		switch rec.Type {
		case "ALIAS":
			// ALIAS is called ANAME on DNS Made Easy
			rec.ChangeType("ANAME", dc.Name)
		case "NS":
			// NS records have fixed TTL on DNS Made Easy and it cannot be changed
			rec.TTL = fixedNameServerRecordTTL
		}
	}

	toReport, create, del, modify, actualChangeCount, err := diff.NewCompat(dc).IncrementalDiff(existingRecords)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	var deleteRecordIds []int
	deleteDescription := []string{"Batch deletion of records:"}
	for _, m := range del {
		originalRecordID := m.Existing.Original.(*recordResponseDataEntry).ID
		deleteRecordIds = append(deleteRecordIds, originalRecordID)
		deleteDescription = append(deleteDescription, m.String())
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

	var createRecords []recordRequestData
	createDescription := []string{"Batch creation of records:"}
	for _, m := range create {
		record := fromRecordConfig(m.Desired)
		createRecords = append(createRecords, *record)
		createDescription = append(createDescription, m.String())
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

	var modifyRecords []recordRequestData
	modifyDescription := []string{"Batch modification of records:"}
	for _, m := range modify {
		originalRecord := m.Existing.Original.(*recordResponseDataEntry)

		record := fromRecordConfig(m.Desired)
		record.ID = originalRecord.ID
		record.GtdLocation = originalRecord.GtdLocation

		modifyRecords = append(modifyRecords, *record)
		modifyDescription = append(modifyDescription, m.String())
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

	existingRecords := make([]*models.RecordConfig, 0, len(records))
	for i := range records {
		// Ignore HTTPRED and SPF records
		if records[i].Type == "HTTPRED" || records[i].Type == "SPF" {
			continue
		}
		existingRecords = append(existingRecords, toRecordConfig(domain, &records[i]))
	}

	for i := range nameServers {
		existingRecords = append(existingRecords, systemNameServerToRecordConfig(domain, nameServers[i]))
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
