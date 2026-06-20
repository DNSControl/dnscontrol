package websupport

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://rest.websupport.sk"
	rowsPerPage    = 1000
)

// nativeRecord matches the WebSupport REST API v2 DNS record shape. The
// `name` field is the fully-qualified record name (e.g. "www.example.com",
// with the apex represented as the bare domain "example.com").
type nativeRecord struct {
	ID       int64  `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      uint32 `json:"ttl,omitempty"`
	Priority *int   `json:"priority,omitempty"`
	Port     *int   `json:"port,omitempty"`
	Weight   *int   `json:"weight,omitempty"`
	Note     string `json:"note,omitempty"`
}

// recordPage is one page of GET /v2/service/{service}/dns/record.
type recordPage struct {
	CurrentPage  int            `json:"currentPage"`
	TotalPages   int            `json:"totalPages"`
	TotalRecords int            `json:"totalRecords"`
	RowsPerPage  int            `json:"rowsPerPage"`
	Data         []nativeRecord `json:"data"`
}

// v1Service is one entry of the v1 service listing, used to map a domain
// name to the numeric service id that the v2 DNS endpoints require.
type v1Service struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type v1ServiceList struct {
	Items []v1Service `json:"items"`
}

type apiError struct {
	HTTPStatus int
	Body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("WEBSUPPORT: bad API response (%d): %s", e.HTTPStatus, e.Body)
}

func isNotFound(err error) bool {
	if e, ok := err.(*apiError); ok {
		return e.HTTPStatus == http.StatusNotFound
	}
	return false
}

// do performs a signed request against the WebSupport REST API.
//
// Authentication is HTTP Basic where the username is the API key and the
// password is a hex-encoded HMAC-SHA1 signature of the canonical string
// "{METHOD} {path} {unix-timestamp}" keyed by the API secret. The signed
// path excludes the query string. A matching date header carries the same
// timestamp in ISO-8601 basic format (GMT); the header is named "X-Date"
// for the v2 API and "Date" for the v1 API.
func (c *websupportProvider) do(dateHeader, method, endpoint string, body, out any, validStatus ...int) error {
	if len(validStatus) == 0 {
		validStatus = []int{http.StatusOK}
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("WEBSUPPORT: marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, reqBody)
	if err != nil {
		return err
	}

	// The signature covers the path only, never the query string.
	signPath := endpoint
	if i := strings.IndexByte(signPath, '?'); i >= 0 {
		signPath = signPath[:i]
	}

	ts := time.Now().Unix()
	canonical := fmt.Sprintf("%s %s %d", method, signPath, ts)
	mac := hmac.New(sha1.New, []byte(c.secret))
	mac.Write([]byte(canonical))
	signature := hex.EncodeToString(mac.Sum(nil))

	req.SetBasicAuth(c.apiKey, signature)
	req.Header.Set("Accept", "application/json")
	req.Header.Set(dateHeader, time.Unix(ts, 0).UTC().Format("20060102T150405Z"))
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if !slices.Contains(validStatus, resp.StatusCode) {
		return &apiError{HTTPStatus: resp.StatusCode, Body: string(respBody)}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("WEBSUPPORT: decode response: %w (body: %s)", err, string(respBody))
		}
	}
	return nil
}

// serviceID resolves a domain name to its numeric WebSupport service id,
// which the v2 DNS endpoints use as the {service} path segment. The v2 API
// has no service-listing endpoint, so this uses the v1 service listing.
func (c *websupportProvider) serviceID(domain string) (int64, error) {
	if id, ok := c.services[domain]; ok {
		return id, nil
	}

	var list v1ServiceList
	if err := c.do("Date", http.MethodGet, "/v1/user/self/service", nil, &list); err != nil {
		return 0, fmt.Errorf("WEBSUPPORT: listing services: %w", err)
	}
	for _, s := range list.Items {
		c.services[s.Name] = s.ID
	}

	id, ok := c.services[domain]
	if !ok {
		return 0, fmt.Errorf("WEBSUPPORT: domain %q is not a service in this account", domain)
	}
	return id, nil
}

func recordBasePath(serviceID int64) string {
	return "/v2/service/" + strconv.FormatInt(serviceID, 10) + "/dns/record"
}

// getAllRecords returns every record in a zone, walking all pages.
func (c *websupportProvider) getAllRecords(serviceID int64) ([]nativeRecord, error) {
	var all []nativeRecord
	page := 1
	for {
		endpoint := fmt.Sprintf("%s?page=%d&rowsPerPage=%d", recordBasePath(serviceID), page, rowsPerPage)
		var p recordPage
		if err := c.do("X-Date", http.MethodGet, endpoint, nil, &p); err != nil {
			return nil, err
		}
		all = append(all, p.Data...)
		if p.TotalPages == 0 || p.CurrentPage >= p.TotalPages || len(p.Data) == 0 {
			break
		}
		page++
	}
	return all, nil
}

func (c *websupportProvider) createRecord(serviceID int64, r nativeRecord) error {
	// The create endpoint returns 204 with no body, so the new record's id is
	// not echoed back. That's fine: dnscontrol re-reads the zone on the next
	// run, and corrections never need the id of a record they just created.
	return c.do("X-Date", http.MethodPost, recordBasePath(serviceID), r, nil,
		http.StatusNoContent, http.StatusCreated, http.StatusOK)
}

func (c *websupportProvider) updateRecord(serviceID, id int64, r nativeRecord) error {
	endpoint := recordBasePath(serviceID) + "/" + strconv.FormatInt(id, 10)
	return c.do("X-Date", http.MethodPut, endpoint, r, nil, http.StatusNoContent, http.StatusOK)
}

func (c *websupportProvider) deleteRecord(serviceID, id int64) error {
	endpoint := recordBasePath(serviceID) + "/" + strconv.FormatInt(id, 10)
	err := c.do("X-Date", http.MethodDelete, endpoint, nil, nil, http.StatusNoContent, http.StatusOK)
	if isNotFound(err) {
		return nil
	}
	return err
}
