package hostingde

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"golang.org/x/net/idna"
)

const endpoint = "%s/api/%s/v1/json/%s"

type hostingdeProvider struct {
	authToken       string
	ownerAccountID  string
	filterAccountID string
	baseURL         string
	nameservers     []string
	defaultSoa      soaValues
}

func (hp *hostingdeProvider) getDomainConfig(domain string) (*domainConfig, error) {
	params := request{
		Filter: &filter{
			Field: "domainName",
			Value: domain,
		},
	}

	resp, err := hp.get("domain", "domainsFind", params)
	if err != nil {
		return nil, fmt.Errorf("error getting domain info: %w", err)
	}

	domainConf := []*domainConfig{}
	if err := json.Unmarshal(resp.Data, &domainConf); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(domainConf) == 0 {
		return nil, fmt.Errorf("could not get domain config: %s", domain)
	}

	return domainConf[0], nil
}

func (hp *hostingdeProvider) createZone(domain string) error {
	t, err := idna.ToASCII(domain)
	if err != nil {
		return err
	}

	records := []*record{}
	for _, ns := range hp.nameservers {
		records = append(records, &record{
			Name:    domain,
			Type:    "NS",
			Content: ns,
			TTL:     86400,
		})
	}

	params := request{
		ZoneConfig: &zoneConfig{
			Name: t,
			Type: "NATIVE",
		},
		Records: records,
	}

	_, err = hp.get("dns", "zoneCreate", params)
	if err != nil {
		return fmt.Errorf("error creating zone: %w", err)
	}
	return nil
}

func (hp *hostingdeProvider) getNameservers(domain string) ([]string, error) {
	t, err := idna.ToASCII(domain)
	if err != nil {
		return nil, err
	}

	domainConf, err := hp.getDomainConfig(t)
	if err != nil {
		return nil, fmt.Errorf("error getting domain config: %w", err)
	}

	nss := []string{}
	for _, ns := range domainConf.Nameservers {
		// Currently does not support glued IP addresses
		if len(ns.IPs) > 0 {
			return nil, fmt.Errorf("domain %s has glued IP addresses which are not supported", domain)
		}

		nss = append(nss, ns.Name)
	}

	return nss, nil
}

func (hp *hostingdeProvider) updateNameservers(nss []string, domain string) func() error {
	return func() error {
		domainConf, err := hp.getDomainConfig(domain)
		if err != nil {
			return err
		}

		nameservers := []nameserver{}
		for _, ns := range nss {
			nameservers = append(nameservers, nameserver{Name: ns})
		}

		domainConf.Nameservers = nameservers

		params := request{
			Domain: domainConf,
		}

		if _, err := hp.get("domain", "domainUpdate", params); err != nil {
			return err
		}
		return nil
	}
}

func (hp *hostingdeProvider) updateZone(zc *zoneConfig, options *dnsSecOptions, create, del, mod diff.Changeset) error {
	toAdd := []*record{}
	for _, c := range create {
		r := recordToNative(c.Desired)
		toAdd = append(toAdd, r)
	}

	toDelete := []*record{}
	for _, d := range del {
		r := recordToNative(d.Existing)
		r.ID = d.Existing.Original.(record).ID
		toDelete = append(toDelete, r)
	}

	toModify := []*record{}
	for _, m := range mod {
		r := recordToNative(m.Desired)
		r.ID = m.Existing.Original.(record).ID
		toModify = append(toModify, r)
	}

	params := request{
		ZoneConfig:      zc,
		RecordsToAdd:    toAdd,
		RecordsToDelete: toDelete,
		RecordsToModify: toModify,
		DNSSECOptions:   options,
	}

	_, err := hp.get("dns", "zoneUpdate", params)
	if err != nil {
		return err
	}
	return nil
}

func (hp *hostingdeProvider) getZone(domain string) (*zone, error) {
	t, err := idna.ToASCII(domain)
	if err != nil {
		return nil, err
	}

	params := request{
		Filter: &filter{
			Field: "ZoneName",
			Value: t,
		},
	}

	resp, err := hp.get("dns", "zonesFind", params)
	if err != nil {
		return nil, fmt.Errorf("could not get zone config: %w", err)
	}

	zones := []*zone{}
	if err := json.Unmarshal(resp.Data, &zones); err != nil {
		return nil, fmt.Errorf("could not parse response: %w", err)
	}

	if len(zones) == 0 {
		return nil, errZoneNotFound
	}

	return zones[0], nil
}

func (hp *hostingdeProvider) getZoneConfig(domain string) (*zoneConfig, error) {
	t, err := idna.ToASCII(domain)
	if err != nil {
		return nil, err
	}

	params := request{
		Filter: &filter{
			Field: "ZoneName",
			Value: t,
		},
	}

	resp, err := hp.get("dns", "zoneConfigsFind", params)
	if err != nil {
		return nil, fmt.Errorf("could not get zone config: %w", err)
	}

	zc := []*zoneConfig{}
	if err := json.Unmarshal(resp.Data, &zc); err != nil {
		return nil, fmt.Errorf("could not parse response: %w", err)
	}

	if len(zc) == 0 {
		return nil, errZoneNotFound
	}

	return zc[0], nil
}

func (hp *hostingdeProvider) getDNSSECOptions(zoneConfigID string) (*dnsSecOptions, error) {
	params := request{
		Filter: &filter{
			Field: "zoneConfigId",
			Value: zoneConfigID,
		},
	}

	resp, err := hp.get("dns", "dnsSecOptionsFind", params)
	if err != nil {
		return nil, fmt.Errorf("could not get dnssec options: %w", err)
	}

	dnsSecOptions := []*dnsSecOptions{}
	if err := json.Unmarshal(resp.Data, &dnsSecOptions); err != nil {
		return nil, fmt.Errorf("could not parse response: %w", err)
	}

	if len(dnsSecOptions) == 0 {
		return nil, nil
	}

	return dnsSecOptions[0], nil
}

func (hp *hostingdeProvider) dnsSecKeyModify(domain string, add []dnsSecEntry, remove []dnsSecEntry) error {
	params := request{
		DomainName: domain,
		Add:        add,
		Remove:     remove,
	}

	_, err := hp.get("domain", "dnsSecKeyModify", params)
	if err != nil {
		return err
	}
	return nil
}

func (hp *hostingdeProvider) get(service, method string, params request) (*responseData, error) {
	params.AuthToken = hp.authToken
	params.OwnerAccountID = hp.ownerAccountID
	reqBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request body: %w", err)
	}

	url := fmt.Sprintf(endpoint, hp.baseURL, service, method)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("could not carry out request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("error occurred: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	respData := &response{}
	if err := json.Unmarshal(bodyBytes, &respData); err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %w", err)
	}
	if len(respData.Errors) > 0 && respData.Status == "error" {
		return nil, fmt.Errorf("%+v", respData.Errors)
	}

	return respData.Response, nil
}

func (hp *hostingdeProvider) getAllZoneConfigs() ([]*zoneConfig, error) {
	params := request{
		Limit: 10000,
	}
	if hp.filterAccountID != "" {
		params.Filter = &filter{
			Field: "accountId",
			Value: hp.filterAccountID,
		}
	}

	resp, err := hp.get("dns", "zoneConfigsFind", params)
	if err != nil {
		return nil, fmt.Errorf("could not get zones: %w", err)
	}

	zc := []*zoneConfig{}
	if err := json.Unmarshal(resp.Data, &zc); err != nil {
		return nil, fmt.Errorf("could not parse response: %w", err)
	}

	return zc, nil
}
