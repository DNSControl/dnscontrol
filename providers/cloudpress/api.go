package cloudpress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

const (
	pageSize = 100

	// Retry policy for rate limiting (429) and transient server errors.
	maxRetries     = 6
	retryBaseDelay = 1 * time.Second
	maxRetryDelay  = 30 * time.Second
)

// zone mirrors a CloudPress dns_zone object. The records and nameservers are
// only populated by the "view zone" endpoint, not by the "list zones" endpoint.
// IDs are UUID strings, not integers.
type zone struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DNSSEC      bool     `json:"dnssec"`
	Nameservers []string `json:"nameservers"`
	Records     []record `json:"records"`
}

// record mirrors a CloudPress dns_zone record object. The numeric fields are
// nullable in the API, so they are modelled as pointers: that lets us send a
// field only when it is meaningful for the record type (and round-trip JSON
// null as "unset" rather than zero).
type record struct {
	ID         string     `json:"id,omitempty"`
	RecordType recordType `json:"record_type"`
	Name       string     `json:"name"`
	Value      string     `json:"value"`
	TTL        uint32     `json:"ttl"`
	Priority   *uint16    `json:"priority,omitempty"`
	Weight     *uint16    `json:"weight,omitempty"`
	Port       *uint16    `json:"port,omitempty"`
	Flags      *uint8     `json:"flags,omitempty"`
	Tag        string     `json:"record_tag,omitempty"`
}

// zonePayload is the strong-parameters envelope CloudPress expects for zone
// write requests, e.g. {"dns_zone": {"name": "example.com"}}.
type zonePayload struct {
	DNSZone map[string]any `json:"dns_zone"`
}

// listZonesResponse wraps the index response: {"dns_zones": [...]}.
type listZonesResponse struct {
	DNSZones []zone `json:"dns_zones"`
}

// zoneResponse wraps the show/create response: {"dns_zone": {...}}.
type zoneResponse struct {
	DNSZone zone `json:"dns_zone"`
}

// recordPayload is the strong-parameters envelope for record write requests:
// {"dns_record": {...}}.
type recordPayload struct {
	DNSRecord *record `json:"dns_record"`
}

type queryParams map[string]string

func (c *cloudpressProvider) findZoneByDomain(domain string) (*zone, error) {
	if c.zones == nil {
		zones, err := c.getAllZones()
		if err != nil {
			return nil, err
		}

		c.zones = make(map[string]*zone, len(zones))
		for _, z := range zones {
			c.zones[z.Name] = z
		}
	}

	z, ok := c.zones[domain]
	if !ok {
		return nil, fmt.Errorf("%q is not a zone in this CLOUDPRESS account", domain)
	}

	return z, nil
}

func (c *cloudpressProvider) getAllZones() ([]*zone, error) {
	var zones []*zone
	page := 1

	for {
		var res listZonesResponse
		query := queryParams{"page": strconv.Itoa(page), "per_page": strconv.Itoa(pageSize)}
		if err := c.request("GET", "/api/dns_zones", query, nil, &res, nil); err != nil {
			return nil, fmt.Errorf("could not fetch zones: %w", err)
		}

		for i := range res.DNSZones {
			zones = append(zones, &res.DNSZones[i])
		}

		// CloudPress wraps the list in {"dns_zones": [...]}; a short page
		// signals the last page.
		if len(res.DNSZones) < pageSize {
			break
		}
		page++
	}

	return zones, nil
}

// getZone fetches a single zone including its nested records and nameservers.
func (c *cloudpressProvider) getZone(zoneID string) (*zone, error) {
	var res zoneResponse
	endpoint := fmt.Sprintf("/api/dns_zones/%s", zoneID)
	if err := c.request("GET", endpoint, nil, nil, &res, nil); err != nil {
		return nil, err
	}
	return &res.DNSZone, nil
}

func (c *cloudpressProvider) createZone(domain string) (*zone, error) {
	var res zoneResponse
	body := zonePayload{DNSZone: map[string]any{"name": domain}}
	headers := map[string]string{}
	if c.accountID != "" {
		headers["X-Auth-Account"] = c.accountID
	}
	err := c.requestWithHeaders("POST", "/api/dns_zones", nil, body, &res, []int{http.StatusOK, http.StatusCreated}, headers)
	if err != nil {
		return nil, err
	}

	z := &res.DNSZone
	c.zones[domain] = z
	return z, nil
}

func (c *cloudpressProvider) createRecord(zoneID string, r *record) error {
	endpoint := fmt.Sprintf("/api/dns_zones/%s/records", zoneID)
	// CloudPress responds with HTTP 200 (not 201) on record creation.
	return c.request("POST", endpoint, nil, recordPayload{DNSRecord: r}, nil, []int{http.StatusOK, http.StatusCreated})
}

func (c *cloudpressProvider) modifyRecord(zoneID, recordID string, r *record) error {
	endpoint := fmt.Sprintf("/api/dns_zones/%s/records/%s", zoneID, recordID)
	// PATCH cannot modify name or record_type; CloudPress ignores those fields
	// and responds with HTTP 202.
	return c.request("PATCH", endpoint, nil, recordPayload{DNSRecord: r}, nil, []int{http.StatusOK, http.StatusAccepted})
}

func (c *cloudpressProvider) deleteRecord(zoneID, recordID string) error {
	endpoint := fmt.Sprintf("/api/dns_zones/%s/records/%s", zoneID, recordID)
	return c.request("DELETE", endpoint, nil, nil, nil, []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent})
}

func (c *cloudpressProvider) setDNSSEC(zoneID string, enabled bool) error {
	endpoint := fmt.Sprintf("/api/dns_zones/%s", zoneID)
	body := zonePayload{DNSZone: map[string]any{"dnssec": enabled}}
	return c.request("PATCH", endpoint, nil, body, nil, []int{http.StatusOK, http.StatusAccepted})
}

func (c *cloudpressProvider) request(method, endpoint string, query queryParams, body, target any, validStatus []int) error {
	return c.requestWithHeaders(method, endpoint, query, body, target, validStatus, nil)
}

func (c *cloudpressProvider) requestWithHeaders(method, endpoint string, query queryParams, body, target any, validStatus []int, headers map[string]string) error {
	if validStatus == nil {
		validStatus = []int{http.StatusOK}
	}

	// Marshal the body once and rebuild a fresh reader for each attempt, since
	// a request body is consumed when sent and must be replayable on retry.
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	var resp *http.Response
	for attempt := 0; ; attempt++ {
		var requestBody io.Reader
		if bodyBytes != nil {
			requestBody = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequest(method, c.baseURL+endpoint, requestBody)
		if err != nil {
			return err
		}

		req.Header.Add("Authorization", "Bearer "+c.apiToken)
		req.Header.Add("Accept", "application/json")
		if bodyBytes != nil {
			req.Header.Add("Content-Type", "application/json")
		}
		for k, v := range headers {
			req.Header.Add(k, v)
		}

		if query != nil {
			q := req.URL.Query()
			for k, v := range query {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		// Retry on rate limiting (429) and transient server errors, honoring
		// the Retry-After header when present.
		if shouldRetry(resp.StatusCode) && attempt < maxRetries {
			wait := retryDelay(resp, attempt)
			_, _ = io.Copy(io.Discard, resp.Body)
			if err := resp.Body.Close(); err != nil {
				printer.Printf("CLOUDPRESS: Could not close response body before retry: %q\n", err)
			}
			printer.Printf("CLOUDPRESS: HTTP %d from %s %s; retrying in %s (attempt %d/%d)\n",
				resp.StatusCode, method, endpoint, wait, attempt+1, maxRetries)
			time.Sleep(wait)
			continue
		}
		break
	}

	cleanup := func() {
		if err := resp.Body.Close(); err != nil {
			printer.Printf("CLOUDPRESS: Could not close response body after API call: %q\n", err)
		}
	}

	if !slices.Contains(validStatus, resp.StatusCode) {
		data, _ := io.ReadAll(resp.Body)
		printer.Println(fmt.Sprintf("CLOUDPRESS: Bad API response for %s %s: %s", method, endpoint, string(data)))
		cleanup()
		return fmt.Errorf("bad status code from CLOUDPRESS: %d not in %v", resp.StatusCode, validStatus)
	}

	if target == nil {
		cleanup()
		return nil
	}

	err := json.NewDecoder(resp.Body).Decode(target)
	cleanup()
	return err
}

// shouldRetry reports whether a response status warrants a retry.
func shouldRetry(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// retryDelay returns how long to wait before the next attempt. It prefers the
// Retry-After header (delay-seconds form) and otherwise uses capped exponential
// backoff.
func retryDelay(resp *http.Response, attempt int) time.Duration {
	if ra := strings.TrimSpace(resp.Header.Get("Retry-After")); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs >= 0 {
			return min(time.Duration(secs)*time.Second, maxRetryDelay)
		}
	}
	return min(retryBaseDelay<<attempt, maxRetryDelay)
}
