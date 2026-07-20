package dnsmadeeasy

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

type dnsMadeEasyProvider struct {
	restAPI *dnsMadeEasyRestAPI
	domains map[string]int
}

func newProvider(apiKey string, secretKey string, sandbox bool, debug bool) *dnsMadeEasyProvider {
	baseURL := baseURLV2_0
	if sandbox {
		baseURL = sandboxBaseURLV2_0
	}

	printer.Printf("Creating DNSMADEEASY provider for %q\n", baseURL)

	return &dnsMadeEasyProvider{
		restAPI: &dnsMadeEasyRestAPI{
			apiKey:    apiKey,
			secretKey: secretKey,
			baseURL:   baseURL,
			httpClient: &http.Client{
				Timeout: time.Minute,
			},
			dumpHTTPRequest:  debug,
			dumpHTTPResponse: debug,
		},
	}
}

func (api *dnsMadeEasyProvider) loadDomains() error {
	if api.domains != nil {
		return nil
	}

	domains := map[string]int{}

	res, err := api.restAPI.multiDomainGet()
	if err != nil {
		return fmt.Errorf("fetching domains from DNSMADEEASY failed: %w", err)
	}

	for _, domain := range res.Data {
		if domain.GtdEnabled {
			return errors.New("fetching domains from DNSMADEEASY failed: domains with GTD enabled are not supported")
		}

		domains[domain.Name] = domain.ID
	}

	api.domains = domains

	return nil
}

func (api *dnsMadeEasyProvider) domainExists(name string) (bool, error) {
	if err := api.loadDomains(); err != nil {
		return false, err
	}

	_, ok := api.domains[name]

	return ok, nil
}

func (api *dnsMadeEasyProvider) findDomainID(name string) (int, error) {
	if err := api.loadDomains(); err != nil {
		return 0, err
	}

	domain, ok := api.domains[name]
	if !ok {
		return 0, fmt.Errorf("domain not found on this DNSMADEEASY account: %q", name)
	}

	return domain, nil
}

func (api *dnsMadeEasyProvider) fetchDomainRecords(domainName string) ([]recordResponseDataEntry, error) {
	domainID, err := api.findDomainID(domainName)
	if err != nil {
		return nil, err
	}

	res, err := api.restAPI.recordGet(domainID)
	if err != nil {
		return nil, fmt.Errorf("fetching records failed: %w", err)
	}

	records := make([]recordResponseDataEntry, 0)
	for _, record := range res.Data {
		if record.GtdLocation != "DEFAULT" {
			return nil, errors.New("fetching records from DNSMADEEASY failed: only records with DEFAULT GTD location are supported")
		}

		records = append(records, record)
	}

	return records, nil
}

func (api *dnsMadeEasyProvider) fetchDomainNameServers(domainName string) ([]string, error) {
	domainID, err := api.findDomainID(domainName)
	if err != nil {
		return nil, err
	}

	res, err := api.restAPI.singleDomainGet(domainID)
	if err != nil {
		return nil, fmt.Errorf("fetching domain from DNSMADEEASY failed: %w", err)
	}

	var nameServers []string
	for i := range res.NameServers {
		nameServers = append(nameServers, res.NameServers[i].Fqdn)
	}

	return nameServers, nil
}

func (api *dnsMadeEasyProvider) createDomain(domain string) error {
	res, err := api.restAPI.singleDomainCreate(singleDomainRequestData{Name: domain})
	if err != nil {
		return err
	}

	api.domains[domain] = res.ID

	return nil
}

func (api *dnsMadeEasyProvider) deleteRecords(domainID int, recordIds []int) error {
	err := api.restAPI.multiRecordDelete(domainID, recordIds)

	return err
}

func (api *dnsMadeEasyProvider) updateRecords(domainID int, records []recordRequestData) error {
	err := api.restAPI.multiRecordUpdate(domainID, records)

	return err
}

func (api *dnsMadeEasyProvider) createRecords(domainID int, records []recordRequestData) error {
	_, err := api.restAPI.multiRecordCreate(domainID, records)

	return err
}
