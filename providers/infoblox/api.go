package infoblox

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
)

type infobloxAPI struct {
	host       string // e.g. "https://grid-master.example.com"
	wapiVer    string // e.g. "2.12"
	view       string // e.g. "default"
	username   string
	password   string
	httpClient *http.Client
}

// wapiResponse is the standard WAPI response wrapper when _return_as_object=1 is used.
type wapiResponse struct {
	Result json.RawMessage `json:"result"`
}

func newInfobloxAPI(host, username, password, wapiVer, view string, tlsSkipVerify bool, caCertPath string) (*infobloxAPI, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if tlsSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate %q: %w", caCertPath, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate from %q", caCertPath)
		}
		tlsConfig.RootCAs = pool
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	return &infobloxAPI{
		host:     strings.TrimRight(host, "/"),
		wapiVer:  wapiVer,
		view:     view,
		username: username,
		password: password,
		httpClient: &http.Client{
			Jar: jar,
			Transport: &http.Transport{
				TLSClientConfig:    tlsConfig,
				DisableCompression: true,
			},
		},
	}, nil
}

func (api *infobloxAPI) baseURL() string {
	return fmt.Sprintf("%s/wapi/v%s", api.host, api.wapiVer)
}

func (api *infobloxAPI) doRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(api.username, api.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("WAPI request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("WAPI %s %s returned HTTP %d: %s", method, url, resp.StatusCode, string(data))
	}

	return data, nil
}

// getZoneAuth looks up a zone_auth object for the given FQDN in the configured view.
// Returns the zone _ref or an error if the zone is not found.
func (api *infobloxAPI) getZoneAuth(fqdn string) (string, error) {
	url := fmt.Sprintf("%s/zone_auth?fqdn=%s&view=%s&_return_as_object=1", api.baseURL(), fqdn, api.view)
	data, err := api.doRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	var resp wapiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to parse zone_auth response: %w", err)
	}

	var zones []struct {
		Ref string `json:"_ref"`
	}
	if err := json.Unmarshal(resp.Result, &zones); err != nil {
		return "", fmt.Errorf("failed to parse zone_auth result: %w", err)
	}

	if len(zones) == 0 {
		return "", fmt.Errorf("zone %q not found in view %q", fqdn, api.view)
	}

	return zones[0].Ref, nil
}

// getZoneDefaultTTL fetches the soa_default_ttl for the given zone.
func (api *infobloxAPI) getZoneDefaultTTL(fqdn string) (uint32, error) {
	url := fmt.Sprintf("%s/zone_auth?fqdn=%s&view=%s&_return_fields=soa_default_ttl&_return_as_object=1", api.baseURL(), fqdn, api.view)
	data, err := api.doRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	var resp wapiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, fmt.Errorf("failed to parse zone TTL response: %w", err)
	}

	var zones []struct {
		SoaDefaultTTL uint32 `json:"soa_default_ttl"`
	}
	if err := json.Unmarshal(resp.Result, &zones); err != nil {
		return 0, fmt.Errorf("failed to parse zone TTL result: %w", err)
	}

	if len(zones) == 0 {
		return 0, fmt.Errorf("zone %q not found in view %q", fqdn, api.view)
	}

	return zones[0].SoaDefaultTTL, nil
}

// getRecords fetches all records of a given type for a zone.
// recType should be the Infoblox record type string (e.g. "a", "aaaa", "cname").
func (api *infobloxAPI) getRecords(recType, zone string) ([]json.RawMessage, error) {
	returnFields := extraReturnFields(recType)
	url := fmt.Sprintf("%s/record:%s?zone=%s&view=%s%s&_max_results=-10000&_return_as_object=1",
		api.baseURL(), recType, zone, api.view, returnFields)

	data, err := api.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var resp wapiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse record:%s response: %w", recType, err)
	}

	var records []json.RawMessage
	if err := json.Unmarshal(resp.Result, &records); err != nil {
		return nil, fmt.Errorf("failed to parse record:%s result: %w", recType, err)
	}

	return records, nil
}

// extraReturnFields returns the additional _return_fields query parameter
// needed for each record type. Most types need ttl and use_ttl appended.
// Some types also need their data fields explicitly requested because
// Infoblox marks them as non-standard.
func extraReturnFields(recType string) string {
	switch recType {
	case "ns":
		// NS records in Infoblox don't have ttl or use_ttl fields.
		return ""
	case "caa":
		// ca_flag, ca_tag, ca_value are non-standard fields.
		return "&_return_fields%2B=ttl,use_ttl,ca_flag,ca_tag,ca_value"
	case "ptr":
		// name is a non-standard field for PTR records.
		return "&_return_fields%2B=ttl,use_ttl,name"
	default:
		return "&_return_fields%2B=ttl,use_ttl"
	}
}

// createRecord creates a new record in Infoblox and returns the _ref of the created object.
func (api *infobloxAPI) createRecord(recType string, body map[string]any) (string, error) {
	url := fmt.Sprintf("%s/record:%s?_return_as_object=1", api.baseURL(), recType)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal record body: %w", err)
	}

	data, err := api.doRequest("POST", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}

	var resp wapiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to parse create response: %w", err)
	}

	var ref string
	if err := json.Unmarshal(resp.Result, &ref); err != nil {
		return "", fmt.Errorf("failed to parse create result ref: %w", err)
	}

	return ref, nil
}

// updateRecord updates an existing record identified by _ref.
func (api *infobloxAPI) updateRecord(ref string, body map[string]any) error {
	url := fmt.Sprintf("%s/%s?_return_as_object=1", api.baseURL(), ref)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal update body: %w", err)
	}

	_, err = api.doRequest("PUT", url, strings.NewReader(string(jsonBody)))
	return err
}

// deleteRecord deletes a record identified by _ref.
func (api *infobloxAPI) deleteRecord(ref string) error {
	url := fmt.Sprintf("%s/%s?_return_as_object=1", api.baseURL(), ref)
	_, err := api.doRequest("DELETE", url, nil)
	return err
}
