package cloudns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

// cloudnsProvider is the handle for the ClouDNS API.
type cloudnsProvider struct {
	creds struct {
		id       string
		password string
		subid    string
	}

	requestLimit *AdaptiveLimiter

	sync.Mutex       // Protects all access to the following fields:
	domainIndex      map[string]string
	nameserversNames []string
}

type requestParams map[string]string

type errorResponse struct {
	Status      string `json:"status"`
	Description string `json:"statusDescription"`
}

type nameserverRecord struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type nameserverResponse []nameserverRecord

type zoneRecord struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Zone   string `json:"zone"`
}

type zoneResponse []zoneRecord

type domainRecord struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Host             string `json:"host"`
	Target           string `json:"record"`
	Priority         string `json:"priority"`
	Weight           string `json:"weight"`
	Port             string `json:"port"`
	Service          string `json:"service"`
	Protocol         string `json:"protocol"`
	TTL              string `json:"ttl"`
	Status           int8   `json:"status"`
	GeodnsCode       string `json:"geodns-location-code,omitempty"`
	CaaFlag          string `json:"caa_flag,omitempty"`
	CaaTag           string `json:"caa_type,omitempty"`
	CaaValue         string `json:"caa_value,omitempty"`
	TlsaUsage        string `json:"tlsa_usage,omitempty"`
	TlsaSelector     string `json:"tlsa_selector,omitempty"`
	TlsaMatchingType string `json:"tlsa_matching_type,omitempty"`
	SshfpAlgorithm   string `json:"algorithm,omitempty"`
	SshfpFingerprint string `json:"fp_type,omitempty"`
	DsKeyTag         string `json:"key_tag,omitempty"`
	DsAlgorithm      string `json:"dsalgorithm,omitempty"`
	DsDigestType     string `json:"digest_type,omitempty"`
	DsDigest         string `json:"dsdigest,omitempty"`
	LocLatDeg        string `json:"lat_deg,omitempty"`
	LocLatMin        string `json:"lat_min,omitempty"`
	LocLatSec        string `json:"lat_sec,omitempty"`
	LocLatDir        string `json:"lat_dir,omitempty"`
	LocLongDeg       string `json:"long_deg,omitempty"`
	LocLongMin       string `json:"long_min,omitempty"`
	LocLongSec       string `json:"long_sec,omitempty"`
	LocLongDir       string `json:"long_dir,omitempty"`
	LocAltitude      string `json:"altitude,omitempty"`
	LocSize          string `json:"size,omitempty"`
	LocHPrecision    string `json:"h_precision,omitempty"`
	LocVPrecision    string `json:"v_precision,omitempty"`
	NaptrOrder       string `json:"order,omitempty"`
	NaptrPreference  string `json:"pref,omitempty"`
	NaptrFlags       string `json:"flag,omitempty"`
	NaptrService     string `json:"params,omitempty"`
	NaptrRegexp      string `json:"regexp,omitempty"`
	NaptrReplacement string `json:"replace,omitempty"`
}

type recordResponse map[string]domainRecord

func (c *cloudnsProvider) fetchAvailableNameservers() ([]string, error) {
	c.Lock()
	defer c.Unlock()

	if c.nameserversNames == nil {
		bodyString, err := c.get("/dns/available-name-servers.json", requestParams{})
		if err != nil {
			return nil, fmt.Errorf("failed fetching available nameservers list from ClouDNS: %w", err)
		}

		var nr nameserverResponse
		if err := json.Unmarshal(bodyString, &nr); err != nil {
			return nil, fmt.Errorf("failed to unmarshal available nameservers list from ClouDNS: %w", err)
		}

		for _, nameserver := range nr {
			if nameserver.Type == "premium" {
				c.nameserversNames = append(c.nameserversNames, nameserver.Name)
			}
		}
	}
	return c.nameserversNames, nil
}

func (c *cloudnsProvider) fetchAvailableTTLValues(domain string) ([]uint32, error) {
	allowedTTLValues := make([]uint32, 0)
	params := requestParams{
		"domain-name": domain,
	}

	bodyString, err := c.get("/dns/get-available-ttl.json", params)
	if err != nil {
		return nil, fmt.Errorf("failed fetching available TTL values list from ClouDNS: %w", err)
	}

	if err := json.Unmarshal(bodyString, &allowedTTLValues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal available TTL values list from ClouDNS: %w", err)
	}
	return allowedTTLValues, nil
}

func (c *cloudnsProvider) idForDomain(name string) (string, bool, error) {
	if err := c.fetchZones(); err != nil {
		return "", false, err
	}

	index, ok := c.domainIndex[name]
	return index, ok, nil
}

func (c *cloudnsProvider) fetchZones() error {
	c.Lock()
	defer c.Unlock()

	if c.domainIndex != nil {
		return nil
	}

	c.domainIndex = map[string]string{}

	rowsPerPage := 100
	page := 1
	for {
		var dr zoneResponse
		params := requestParams{
			"page":          strconv.Itoa(page),
			"rows-per-page": strconv.Itoa(rowsPerPage),
		}
		endpoint := "/dns/list-zones.json"
		bodyString, err := c.get(endpoint, params)
		if err != nil {
			return fmt.Errorf("failed fetching domain list from ClouDNS: %w", err)
		}
		if err := json.Unmarshal(bodyString, &dr); err != nil {
			return fmt.Errorf("failed to unmarshal domain list from ClouDNS: %w", err)
		}

		for _, domain := range dr {
			c.domainIndex[domain.Name] = domain.Name
		}
		if len(dr) < rowsPerPage {
			break
		}
		page++
	}

	return nil
}

func (c *cloudnsProvider) createDomain(domain string) error {
	params := requestParams{
		"domain-name": domain,
		"zone-type":   "master",
	}
	if _, err := c.get("/dns/register.json", params); err != nil {
		return fmt.Errorf("failed create domain (ClouDNS): %w", err)
	}
	c.Lock()
	defer c.Unlock()
	if c.domainIndex != nil {
		c.domainIndex[domain] = domain
	}
	return nil
}

func (c *cloudnsProvider) createRecord(domainID string, rec requestParams) error {
	rec["domain-name"] = domainID
	if _, err := c.get("/dns/add-record.json", rec); err != nil { // here we add record
		return fmt.Errorf("failed create record (ClouDNS): %w", err)
	}
	return nil
}

func (c *cloudnsProvider) deleteRecord(domainID string, recordID string) error {
	params := requestParams{
		"domain-name": domainID,
		"record-id":   recordID,
	}
	if _, err := c.get("/dns/delete-record.json", params); err != nil {
		return fmt.Errorf("failed delete record (ClouDNS): %w", err)
	}
	return nil
}

func (c *cloudnsProvider) modifyRecord(domainID string, recordID string, rec requestParams) error {
	rec["domain-name"] = domainID
	rec["record-id"] = recordID
	if _, err := c.get("/dns/mod-record.json", rec); err != nil {
		return fmt.Errorf("failed update (ClouDNS): %w", err)
	}
	return nil
}

func (c *cloudnsProvider) getRecords(id string) ([]domainRecord, error) {
	params := requestParams{"domain-name": id}

	bodyString, err := c.get("/dns/records.json", params)
	if err != nil {
		return nil, fmt.Errorf("failed fetching record list from ClouDNS: %w", err)
	}

	var dr recordResponse
	if err := json.Unmarshal(bodyString, &dr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record list from ClouDNS: %w", err)
	}

	var records []domainRecord
	for _, rec := range dr {
		records = append(records, rec)
	}
	return records, nil
}

func (c *cloudnsProvider) isDnssecEnabled(id string) (bool, error) {
	params := requestParams{"domain-name": id}

	bodyString, err := c.get("/dns/get-dnssec-ds-records.json", params)
	if err != nil {
		// DNSSEC disabled is indicated by an error fetching the DS records.
		var errResp errorResponse
		err = json.Unmarshal(bodyString, &errResp)
		if err == nil {
			if errResp.Description == "The DNSSEC is not active." {
				return false, nil
			}
			return false, fmt.Errorf("failed fetching DS records from ClouDNS: %s", errResp.Description)
		}
	}

	return true, nil
}

func (c *cloudnsProvider) setDnssec(id string, enabled bool) error {
	params := requestParams{"domain-name": id}

	var endpoint string
	if enabled {
		endpoint = "/dns/activate-dnssec.json"
	} else {
		endpoint = "/dns/deactivate-dnssec.json"
	}

	_, err := c.get(endpoint, params)
	if err != nil {
		return fmt.Errorf("failed setting DNSSEC at ClouDNS: %w", err)
	}

	return nil
}

func (c *cloudnsProvider) getNameservers(domain string) ([]string, error) {
	params := requestParams{"domain-name": domain}

	bodyString, err := c.get("/domains/get-nameservers.json", params)
	if err != nil {
		return nil, fmt.Errorf("failed fetching nameservers from ClouDNS: %w", err)
	}

	// The API response might be an array of strings, or an object with string keys "1", "2", etc.
	// And sometimes it contains values that are blank strings, which we don't want to include in the result.

	// Try as array first
	var arr []string
	errFromArray := json.Unmarshal(bodyString, &arr)
	if errFromArray == nil {
		// Filter out empty strings
		nameservers := make([]string, 0, len(arr))
		for _, ns := range arr {
			if ns != "" {
				nameservers = append(nameservers, ns)
			}
		}
		return nameservers, nil
	}

	// Try as map
	var nr map[string]string
	if err := json.Unmarshal(bodyString, &nr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal nameservers from ClouDNS: when attempting object %w, when attempting array %w", err, errFromArray)
	}

	nameservers := make([]string, 0, len(nr))
	for _, ns := range nr {
		if ns != "" {
			nameservers = append(nameservers, ns)
		}
	}

	return nameservers, nil
}

func (c *cloudnsProvider) setNameservers(domain string, nameservers []string) error {
	q := url.Values{}
	q.Add("domain-name", domain)
	for _, ns := range nameservers {
		q.Add("nameservers[]", ns)
	}

	if _, err := c.getWithQuery("/domains/set-nameservers.json", q); err != nil {
		return fmt.Errorf("failed setting nameservers at ClouDNS: %w", err)
	}
	return nil
}

func (c *cloudnsProvider) get(endpoint string, params requestParams) ([]byte, error) {
	q := url.Values{}
	for pName, pValue := range params {
		q.Add(pName, pValue)
	}
	return c.getWithQuery(endpoint, q)
}

func (c *cloudnsProvider) getWithQuery(endpoint string, q url.Values) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, "https://api.cloudns.net"+endpoint, nil)

	// TODO: Support  sub-auth-user https://asia.cloudns.net/wiki/article/42/
	// Add auth params
	q.Add("auth-id", c.creds.id)
	q.Add("auth-password", c.creds.password)
	q.Add("sub-auth-id", c.creds.subid)

	req.URL.RawQuery = q.Encode()

	// ClouDNS has an undocumented rate limit
	// But the API does not provide a different status code (it's a 200) nor rate limit headers.
	// The response is {"status":"Failed","statusDescription":"Request blocked. 84.86.84.86 is sending more than 20 requests per second."}

retry:
	if err := c.requestLimit.Wait(context.Background()); err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}

	bodyString, _ := io.ReadAll(resp.Body)

	// Got error from API ?
	var errResp errorResponse
	err = json.Unmarshal(bodyString, &errResp)
	if err == nil {
		if errResp.Status == "Failed" {
			// If this is a rate limit error
			if strings.Contains(errResp.Description, "Request blocked.") && strings.Contains(errResp.Description, "is sending more than") {

				printer.Warnf("ClouDNS API rate limited, pausing API calls...\n")
				c.requestLimit.NotifyRateLimited()

				goto retry
			}

			// For debug only - req.URL.RequestURI() contains the authentication params:
			// return bodyString, fmt.Errorf("ClouDNS API error: %s URL:%s%s ", errResp.Description, req.Host, req.URL.RequestURI())
			return bodyString, fmt.Errorf("ClouDNS API error: %s", errResp.Description)
		}
	}

	return bodyString, nil
}

func fixTTL(allowedTTLValues []uint32, ttl uint32) uint32 {
	// if the TTL is larger than the largest allowed value, return the largest allowed value
	if ttl > allowedTTLValues[len(allowedTTLValues)-1] {
		return allowedTTLValues[len(allowedTTLValues)-1]
	}

	for _, v := range allowedTTLValues {
		if v >= ttl {
			return v
		}
	}

	return allowedTTLValues[0]
}
