package netlify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const baseURL = "https://api.netlify.com/api/v1"

type dnsRecord struct {
	Hostname  string `json:"hostname,omitempty"`
	Type      string `json:"type,omitempty"`
	TTL       int64  `json:"ttl,omitempty"`
	Priority  int64  `json:"priority,omitempty"`
	Flag      int64  `json:"flag,omitempty"`
	Weight    uint16 `json:"weight,omitempty"`
	Port      uint16 `json:"port,omitempty"`
	Tag       string `json:"tag,omitempty"`
	ID        string `json:"id,omitempty"`
	SiteID    string `json:"site_id,omitempty"`
	DNSZoneID string `json:"dns_zone_id,omitempty"`
	Managed   bool   `json:"managed,omitempty"`
	Value     string `json:"value,omitempty"`
}

type dnsZone struct {
	AccountID            string       `json:"account_id,omitempty"`
	AccountName          string       `json:"account_name,omitempty"`
	AccountSlug          string       `json:"account_slug,omitempty"`
	CreatedAt            string       `json:"created_at,omitempty"`
	Dedicated            bool         `json:"dedicated,omitempty"`
	DNSServers           []string     `json:"dns_servers"`
	Domain               string       `json:"domain,omitempty"`
	Errors               []string     `json:"errors"`
	ID                   string       `json:"id,omitempty"`
	IPV6Enabled          bool         `json:"ipv6_enabled,omitempty"`
	Name                 string       `json:"name,omitempty"`
	Records              []*dnsRecord `json:"records"`
	SiteID               string       `json:"site_id,omitempty"`
	SupportedRecordTypes []string     `json:"supported_record_types"`
	UpdatedAt            string       `json:"updated_at,omitempty"`
	UserID               string       `json:"user_id,omitempty"`
}

type dnsRecordCreate struct {
	Flag     int64  `json:"flag"`
	Hostname string `json:"hostname,omitempty"`
	Port     int64  `json:"port,omitempty"`
	Priority int64  `json:"priority"`
	Tag      string `json:"tag,omitempty"`
	TTL      int64  `json:"ttl,omitempty"`
	Type     string `json:"type,omitempty"`
	Value    string `json:"value,omitempty"`
	Weight   int64  `json:"weight"`
}

func (n *netlifyProvider) getDNSZones() ([]*dnsZone, error) {
	reqURL := baseURL + "/dns_zones"

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+n.apiToken)

	if n.accountSlug != "" {
		q := req.URL.Query()
		q.Add("account_slug", n.accountSlug)
		req.URL.RawQuery = q.Encode()
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	dnsZones := make([]*dnsZone, 0)

	err = json.NewDecoder(res.Body).Decode(&dnsZones)
	if err != nil {
		return nil, err
	}

	return dnsZones, nil
}

func (n *netlifyProvider) getDNSRecords(zoneID string) ([]*dnsRecord, error) {
	reqURL := fmt.Sprintf("%s/dns_zones/%s/dns_records", baseURL, zoneID)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+n.apiToken)

	if n.accountSlug != "" {
		q := req.URL.Query()
		q.Add("account_slug", n.accountSlug)
		req.URL.RawQuery = q.Encode()
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	records := make([]*dnsRecord, 0)

	err = json.NewDecoder(res.Body).Decode(&records)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (n *netlifyProvider) deleteDNSRecord(zoneID string, recordID string) error {
	reqURL := fmt.Sprintf("%s/dns_zones/%s/dns_records/%s", baseURL, zoneID, recordID)

	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+n.apiToken)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

func (n *netlifyProvider) createDNSRecord(zoneID string, rec *dnsRecordCreate) (*dnsRecord, error) {
	reqURL := fmt.Sprintf("%s/dns_zones/%s/dns_records", baseURL, zoneID)

	data, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+n.apiToken)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	record := &dnsRecord{}

	err = json.NewDecoder(res.Body).Decode(record)
	if err != nil {
		return nil, err
	}

	return record, nil
}
