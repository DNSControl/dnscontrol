package bunnydns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"slices"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

const (
	baseURL  = "https://api.bunny.net"
	pageSize = 100
)

type zone struct {
	ID          int64  `json:"Id"`
	Domain      string `json:"Domain"`
	Nameserver1 string `json:"Nameserver1"`
	Nameserver2 string `json:"Nameserver2"`
	HasDNSSEC   bool   `json:"DnsSecEnabled"`
}

func (zone *zone) Nameservers() []string {
	return []string{zone.Nameserver1, zone.Nameserver2}
}

type record struct {
	ID         int64      `json:"Id,omitempty"`
	Type       recordType `json:"Type"`
	Name       string     `json:"Name"`
	Value      string     `json:"Value"`
	Disabled   bool       `json:"Disabled"`
	TTL        uint32     `json:"Ttl"`
	Flags      uint8      `json:"Flags"`
	Priority   uint16     `json:"Priority"`
	Weight     uint16     `json:"Weight"`
	Port       uint16     `json:"Port"`
	Tag        string     `json:"Tag"`
	PullZoneID int64      `json:"PullZoneId,omitempty"`
	LinkName   string     `json:"LinkName,omitempty"`
}

type listZonesResponse struct {
	Items        []zone `json:"Items"`
	TotalItems   int32  `json:"TotalItems"`
	HasMoreItems bool   `json:"HasMoreItems"`
}

type getZoneResponse struct {
	zone
	Records []record `json:"Records"`
}

type queryParams map[string]string

func (b *bunnydnsProvider) getImplicitRecordConfigs(zone *zone) (models.Records, error) {
	nameservers := zone.Nameservers()
	records := make(models.Records, 0, len(nameservers))

	// NS records on the zone apex must be implicitly added, as Bunny DNS does not expose them via API
	for _, ns := range nameservers {
		rc := &models.RecordConfig{
			Type:     "NS",
			Original: &record{},
		}
		rc.SetLabelFromFQDN(zone.Domain, zone.Domain)
		if err := rc.SetTarget(ns + "."); err != nil {
			return nil, err
		}

		records = append(records, rc)
	}

	return records, nil
}

func (b *bunnydnsProvider) findZoneByDomain(domain string) (*zone, error) {
	if b.zones == nil {
		zones, err := b.getAllZones()
		if err != nil {
			return nil, err
		}

		b.zones = make(map[string]*zone, len(zones))
		for _, zone := range zones {
			b.zones[zone.Domain] = zone
		}
	}

	zone, ok := b.zones[domain]
	if !ok {
		return nil, fmt.Errorf("%q is not a zone in this BUNNY_DNS account", domain)
	}

	return zone, nil
}

func (b *bunnydnsProvider) getAllZones() ([]*zone, error) {
	var zones []*zone
	page := 1

	for {
		res := listZonesResponse{}
		query := queryParams{"page": strconv.Itoa(page), "perPage": strconv.Itoa(pageSize)}
		if err := b.request("GET", "/dnszone", query, nil, &res, nil); err != nil {
			return nil, fmt.Errorf("could not fetch zones: %w", err)
		}

		if zones == nil {
			zones = make([]*zone, 0, res.TotalItems)
		}
		for i := range res.Items {
			zones = append(zones, &res.Items[i])
		}

		if !res.HasMoreItems {
			break
		}
		page++
	}

	return zones, nil
}

func (b *bunnydnsProvider) createZone(domain string) (*zone, error) {
	zone := &zone{}
	body := map[string]string{"domain": domain}
	err := b.request("POST", "/dnszone", nil, body, &zone, []int{http.StatusCreated})
	if err != nil {
		return nil, err
	}

	b.zones[domain] = zone
	return zone, nil
}

func (b *bunnydnsProvider) getAllRecords(zoneID int64) ([]*record, error) {
	zone := &getZoneResponse{}
	err := b.request("GET", fmt.Sprintf("/dnszone/%d", zoneID), nil, nil, zone, nil)
	if err != nil {
		return nil, err
	}

	records := make([]*record, 0, len(zone.Records))
	for i := range zone.Records {
		records = append(records, &zone.Records[i])
	}

	return records, nil
}

func (b *bunnydnsProvider) createRecord(zoneID int64, r *record) error {
	url := fmt.Sprintf("/dnszone/%d/records", zoneID)
	return b.request("PUT", url, nil, r, nil, []int{http.StatusCreated})
}

func (b *bunnydnsProvider) modifyRecord(zoneID int64, recordID int64, r *record) error {
	url := fmt.Sprintf("/dnszone/%d/records/%d", zoneID, recordID)
	return b.request("POST", url, nil, r, nil, []int{http.StatusNoContent})
}

func (b *bunnydnsProvider) deleteRecord(zoneID, recordID int64) error {
	url := fmt.Sprintf("/dnszone/%d/records/%d", zoneID, recordID)
	return b.request("DELETE", url, nil, nil, nil, []int{http.StatusNoContent})
}

func (b *bunnydnsProvider) enableDNSSEC(zoneID int64) error {
	url := fmt.Sprintf("/dnszone/%d/dnssec", zoneID)
	return b.request("POST", url, nil, nil, nil, []int{http.StatusOK})
}

func (b *bunnydnsProvider) disableDNSSEC(zoneID int64) error {
	url := fmt.Sprintf("/dnszone/%d/dnssec", zoneID)
	return b.request("DELETE", url, nil, nil, nil, []int{http.StatusOK})
}

func (b *bunnydnsProvider) request(method, endpoint string, query queryParams, body, target any, validStatus []int) error {
	if validStatus == nil {
		validStatus = []int{http.StatusOK}
	}

	var requestBody io.Reader
	if body != nil {
		requestBodyJSON, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestBody = bytes.NewBuffer(requestBodyJSON)
	}

	req, err := http.NewRequest(method, baseURL+endpoint, requestBody)
	if err != nil {
		return err
	}

	req.Header.Add("AccessKey", b.apiKey)
	if requestBody != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	if query != nil {
		q := req.URL.Query()
		for k, v := range query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	cleanup := func() {
		if err := resp.Body.Close(); err != nil {
			printer.Printf("BUNNY_DNS: Could not close response body after API call: %q\n", err)
		}
	}

	if !slices.Contains(validStatus, resp.StatusCode) {
		data, _ := io.ReadAll(resp.Body)
		printer.Println(fmt.Sprintf("BUNNY_DNS: Bad API response for %s %s: %s", method, endpoint, string(data)))
		cleanup()
		return fmt.Errorf("bad status code from BUNNY_DNS: %d not in %v", resp.StatusCode, validStatus)
	}

	if target == nil {
		cleanup()
		return nil
	}

	err = json.NewDecoder(resp.Body).Decode(target)
	cleanup()
	return err
}
