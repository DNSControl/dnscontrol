package unifi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	cloudAPIBase = "https://api.ui.com"
)

// API version constants.
const (
	APIVersionAuto   = "auto"
	APIVersionNew    = "new"
	APIVersionLegacy = "legacy"
)

// unifiClient handles HTTP communication with the UniFi API.
type unifiClient struct {
	host       string // Local: "https://10.19.80.1", Cloud: empty
	consoleID  string // Cloud access: console ID (e.g., "28704E24...:1008810555")
	apiKey     string
	site       string
	apiVersion string // "auto", "new", or "legacy"
	siteID     string // Site UUID for new API (fetched lazily)
	skipTLS    bool
	debug      bool
	httpClient *http.Client
	// Cached API availability (for auto mode)
	newAPIAvailable    *bool
	legacyAPIAvailable *bool
}

// newClient creates a new UniFi API client.
func newClient(host, consoleID, apiKey, site, apiVersion string, skipTLS, debug bool) *unifiClient {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipTLS,
		},
	}

	return &unifiClient{
		host:       strings.TrimRight(host, "/"),
		consoleID:  consoleID,
		apiKey:     apiKey,
		site:       site,
		apiVersion: apiVersion,
		skipTLS:    skipTLS,
		debug:      debug,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

// isCloudAccess returns true if using cloud API access.
func (c *unifiClient) isCloudAccess() bool {
	return c.consoleID != ""
}

// buildURL constructs the full URL for an API endpoint.
// Local:  https://{host}/proxy/network/v2/api/site/{site}/static-dns
// Cloud:  https://api.ui.com/v1/connector/consoles/{consoleID}/proxy/network/v2/api/site/{site}/static-dns
func (c *unifiClient) buildURL(path string) string {
	if c.isCloudAccess() {
		return fmt.Sprintf("%s/v1/connector/consoles/%s/proxy/network%s",
			cloudAPIBase, c.consoleID, path)
	}
	return fmt.Sprintf("%s/proxy/network%s", c.host, path)
}

// do executes an HTTP request to the UniFi API.
func (c *unifiClient) do(method, path string, body any) ([]byte, error) {
	url := c.buildURL(path)

	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		rdr = bytes.NewReader(b)
		if c.debug {
			fmt.Printf("[UNIFI] [DEBUG] %s %s\nPayload: %s\n", method, url, string(b))
		}
	} else if c.debug {
		fmt.Printf("[UNIFI] [DEBUG] %s %s\n", method, url)
	}

	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if c.debug {
		fmt.Printf("[UNIFI] [DEBUG] Response (%d): %s\n", resp.StatusCode, string(respBytes))
	}

	if resp.StatusCode >= 300 {
		return respBytes, fmt.Errorf("[UNIFI] %s %s -> %d: %s",
			method, path, resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}

	return respBytes, nil
}

// getRecordsLegacy fetches all static DNS records using the OLD API.
func (c *unifiClient) getRecordsLegacy() ([]legacyDNSRecord, error) {
	path := fmt.Sprintf("/v2/api/site/%s/static-dns", c.site)

	respBytes, err := c.do("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch records: %w", err)
	}

	var records []legacyDNSRecord
	if err := json.Unmarshal(respBytes, &records); err != nil {
		return nil, fmt.Errorf("failed to parse records: %w", err)
	}

	return records, nil
}

// createRecordLegacy creates a new DNS record using the OLD API.
func (c *unifiClient) createRecordLegacy(r map[string]any) (*legacyDNSRecord, error) {
	path := fmt.Sprintf("/v2/api/site/%s/static-dns", c.site)

	respBytes, err := c.do("POST", path, r)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	var created legacyDNSRecord
	if err := json.Unmarshal(respBytes, &created); err != nil {
		return nil, fmt.Errorf("failed to parse created record: %w", err)
	}

	return &created, nil
}

// deleteRecordLegacy deletes a DNS record using the OLD API.
func (c *unifiClient) deleteRecordLegacy(id string) error {
	path := fmt.Sprintf("/v2/api/site/%s/static-dns/%s", c.site, id)

	_, err := c.do("DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete record %s: %w", id, err)
	}

	return nil
}

// updateRecordLegacy updates a record by deleting and recreating it (OLD API has no PUT).
func (c *unifiClient) updateRecordLegacy(id string, r map[string]any) (*legacyDNSRecord, error) {
	// Delete old record
	if err := c.deleteRecordLegacy(id); err != nil {
		return nil, fmt.Errorf("update failed during delete: %w", err)
	}

	// Create new record
	created, err := c.createRecordLegacy(r)
	if err != nil {
		return nil, fmt.Errorf("update failed during create: %w", err)
	}

	return created, nil
}

// ============================================================================
// NEW API Methods (Network 10.1+)
// Base: /proxy/network/integration/v1/sites/{siteId}/dns/policies
// ============================================================================

// sitesResponse wraps the response from the sites endpoint.
type sitesResponse struct {
	Data []siteInfo `json:"data"`
}

// getSiteID fetches the site UUID for the new API.
// The new API uses site UUIDs instead of site names.
func (c *unifiClient) getSiteID() (string, error) {
	if c.siteID != "" {
		return c.siteID, nil
	}

	path := "/integration/v1/sites"
	respBytes, err := c.do("GET", path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch sites: %w", err)
	}

	// Try parsing as wrapped response first ({"data": [...]})
	var sitesResp sitesResponse
	if err := json.Unmarshal(respBytes, &sitesResp); err == nil && len(sitesResp.Data) > 0 {
		// Find site by internalReference (e.g., "default") or name
		for _, s := range sitesResp.Data {
			if s.InternalReference == c.site || strings.EqualFold(s.Name, c.site) {
				c.siteID = s.ID
				return c.siteID, nil
			}
		}
		return "", fmt.Errorf("site '%s' not found", c.site)
	}

	// Try parsing as direct array
	var sites []siteInfo
	if err := json.Unmarshal(respBytes, &sites); err != nil {
		return "", fmt.Errorf("failed to parse sites: %w", err)
	}

	// Find site by internalReference (e.g., "default") or name
	for _, s := range sites {
		if s.InternalReference == c.site || strings.EqualFold(s.Name, c.site) {
			c.siteID = s.ID
			return c.siteID, nil
		}
	}

	return "", fmt.Errorf("site '%s' not found", c.site)
}

// dnsPoliciesPageLimit is the number of records to request per page when listing DNS policies.
const dnsPoliciesPageLimit = 200

// getRecordsNew fetches all DNS policy records using the NEW API, paginating as needed.
func (c *unifiClient) getRecordsNew() ([]dnsPolicyRecord, error) {
	siteID, err := c.getSiteID()
	if err != nil {
		return nil, err
	}

	var allRecords []dnsPolicyRecord
	offset := 0
	for {
		path := fmt.Sprintf("/integration/v1/sites/%s/dns/policies?offset=%d&limit=%d", siteID, offset, dnsPoliciesPageLimit)

		respBytes, err := c.do("GET", path, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch records: %w", err)
		}

		var response dnsPolicyResponse
		if err := json.Unmarshal(respBytes, &response); err != nil {
			return nil, fmt.Errorf("failed to parse records: %w", err)
		}

		allRecords = append(allRecords, response.Data...)

		if len(response.Data) < dnsPoliciesPageLimit {
			break
		}
		offset += dnsPoliciesPageLimit
	}

	return allRecords, nil
}

// createRecordNew creates a new DNS record using the NEW API.
func (c *unifiClient) createRecordNew(r *dnsPolicyRecord) (*dnsPolicyRecord, error) {
	siteID, err := c.getSiteID()
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/integration/v1/sites/%s/dns/policies", siteID)

	respBytes, err := c.do("POST", path, r)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	var created dnsPolicyRecord
	if err := json.Unmarshal(respBytes, &created); err != nil {
		return nil, fmt.Errorf("failed to parse created record: %w", err)
	}

	return &created, nil
}

// updateRecordNew updates an existing DNS record using the NEW API (native PUT support).
func (c *unifiClient) updateRecordNew(id string, r *dnsPolicyRecord) (*dnsPolicyRecord, error) {
	siteID, err := c.getSiteID()
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/integration/v1/sites/%s/dns/policies/%s", siteID, id)

	// The ID is in the URL path; the API rejects it if included in the request body.
	r.ID = ""

	respBytes, err := c.do("PUT", path, r)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}

	var updated dnsPolicyRecord
	if err := json.Unmarshal(respBytes, &updated); err != nil {
		return nil, fmt.Errorf("failed to parse updated record: %w", err)
	}

	return &updated, nil
}

// deleteRecordNew deletes a DNS record using the NEW API.
func (c *unifiClient) deleteRecordNew(id string) error {
	siteID, err := c.getSiteID()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/integration/v1/sites/%s/dns/policies/%s", siteID, id)

	_, err = c.do("DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete record %s: %w", id, err)
	}

	return nil
}

// ============================================================================
// Auto-detection and Unified API Methods
// ============================================================================

// detectAPIAvailability probes both APIs to determine which are available.
func (c *unifiClient) detectAPIAvailability() {
	if c.newAPIAvailable != nil && c.legacyAPIAvailable != nil {
		return // Already detected
	}

	// Try new API first
	newAvailable := false
	if _, err := c.getSiteID(); err == nil {
		// Site ID fetched successfully, try to get records
		siteID := c.siteID
		path := fmt.Sprintf("/integration/v1/sites/%s/dns/policies?limit=1", siteID)
		if _, err := c.do("GET", path, nil); err == nil {
			newAvailable = true
		}
	}
	c.newAPIAvailable = &newAvailable

	// Try legacy API
	legacyAvailable := false
	path := fmt.Sprintf("/v2/api/site/%s/static-dns", c.site)
	if _, err := c.do("GET", path, nil); err == nil {
		legacyAvailable = true
	}
	c.legacyAPIAvailable = &legacyAvailable

	if c.debug {
		fmt.Printf("[UNIFI] [DEBUG] API availability - New: %v, Legacy: %v\n", newAvailable, legacyAvailable)
	}
}

// useNewAPI returns true if the new API should be used based on config and availability.
func (c *unifiClient) useNewAPI() bool {
	switch c.apiVersion {
	case APIVersionNew:
		return true
	case APIVersionLegacy:
		return false
	case APIVersionAuto:
		c.detectAPIAvailability()
		if c.newAPIAvailable != nil && *c.newAPIAvailable {
			return true
		}
		return false
	default:
		return false
	}
}

// getRecords fetches records using the appropriate API based on configuration.
// In auto mode, tries new API first, then falls back to legacy.
func (c *unifiClient) getRecords() ([]any, bool, error) {
	if c.apiVersion == APIVersionAuto {
		c.detectAPIAvailability()

		// Try new API first
		if c.newAPIAvailable != nil && *c.newAPIAvailable {
			records, err := c.getRecordsNew()
			if err == nil {
				result := make([]any, len(records))
				for i := range records {
					result[i] = &records[i]
				}
				return result, true, nil
			}
			if c.debug {
				fmt.Printf("[UNIFI] [DEBUG] New API failed, falling back to legacy: %v\n", err)
			}
		}

		// Fall back to legacy
		if c.legacyAPIAvailable != nil && *c.legacyAPIAvailable {
			records, err := c.getRecordsLegacy()
			if err == nil {
				result := make([]any, len(records))
				for i := range records {
					result[i] = &records[i]
				}
				return result, false, nil
			}
		}

		return nil, false, fmt.Errorf("no API available")
	}

	if c.useNewAPI() {
		records, err := c.getRecordsNew()
		if err != nil {
			return nil, true, err
		}
		result := make([]any, len(records))
		for i := range records {
			result[i] = &records[i]
		}
		return result, true, nil
	}

	records, err := c.getRecordsLegacy()
	if err != nil {
		return nil, false, err
	}
	result := make([]any, len(records))
	for i := range records {
		result[i] = &records[i]
	}
	return result, false, nil
}
