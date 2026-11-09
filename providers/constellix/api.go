package constellix

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/version"
)

const (
	// defaultAPIURL is the base URL of the v4 API.
	defaultAPIURL = "https://api.dns.constellix.com/v4"

	// pageSize is the largest page the list endpoints return.
	pageSize = 100

	// maxErrorBody bounds how much of a body that is not the API's own error
	// document goes into an error.
	maxErrorBody = 512

	// The rate limit is a window of 180 units per 60 seconds shared by every
	// client of the account. A 429 carries no Retry-After; X-RateLimit-Reset
	// is the number of seconds until the window resets. The margin covers the
	// rounding of that header, and the fallback is one whole window.
	rateLimitMargin = time.Second
	rateLimitPause  = 60 * time.Second

	// maxRateLimitWait bounds how long one request waits for the rate limit
	// in total (five windows) before it fails.
	maxRateLimitWait = 5 * time.Minute
)

type apiClient struct {
	baseURL    string
	apiKey     string
	secretKey  string
	userAgent  string
	httpClient *http.Client

	// now and sleep are time.Now and time.Sleep, replaced in tests: the
	// token embeds the time, and a rate-limit pause must not cost the
	// wall-clock time of the window.
	now   func() time.Time
	sleep func(time.Duration)

	// The gate: once the API says the window is used up, every request of
	// this client waits until pauseUntil.
	mu         sync.Mutex
	pauseUntil time.Time
}

// flexID is an ID the API sends as a JSON number in some places and as a
// string in others (a zone's id is a number in GET /domains and a string in
// GET /domains/{id}).
type flexID string

// UnmarshalJSON accepts a JSON number or a JSON string.
func (id *flexID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*id = flexID(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("id %s is neither a number nor a string", b)
	}
	*id = flexID(n.String())
	return nil
}

// listResponse is the envelope of every list endpoint.
type listResponse[T any] struct {
	Data []T `json:"data"`
	Meta struct {
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	} `json:"meta"`
}

// itemResponse is the envelope of every endpoint that returns one object.
type itemResponse[T any] struct {
	Data T `json:"data"`
}

// apiDomain is a zone as returned by GET /domains and GET /domains/{id}.
// Template is null unless the zone is linked to a template.
type apiDomain struct {
	ID          flexID          `json:"id"`
	Name        string          `json:"name"`
	Nameservers []string        `json:"nameservers"`
	Template    json.RawMessage `json:"template"`
}

// apiRecord is a record object as returned by GET /domains/{id}/records: one
// name + type with a list of values. Name is relative to the zone, "" at the
// apex. Value is kept raw because its shape depends on the type and the mode
// (an object for HTTP redirects and failover records); see checkRecord.
type apiRecord struct {
	ID           flexID          `json:"id"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	TTL          uint32          `json:"ttl"`
	Mode         string          `json:"mode,omitempty"`
	Region       string          `json:"region,omitempty"`
	IPFilter     json.RawMessage `json:"ipfilter,omitempty"`
	GeoProximity json.RawMessage `json:"geoproximity,omitempty"`
	Enabled      *bool           `json:"enabled,omitempty"`
	Value        json.RawMessage `json:"value"`
}

// apiValue is one value of a standard-mode record. Which fields are used
// depends on the record type.
type apiValue struct {
	Enabled  *bool  `json:"enabled"`
	Value    string `json:"value"`    // A, AAAA, CNAME, ANAME, TXT
	Host     string `json:"host"`     // NS, SRV
	System   string `json:"system"`   // PTR
	Server   string `json:"server"`   // MX
	Priority uint16 `json:"priority"` // MX, SRV
	Weight   uint16 `json:"weight"`   // SRV
	Port     uint16 `json:"port"`     // SRV
	Flags    uint8  `json:"flags"`    // CAA
	Tag      string `json:"tag"`      // CAA
	Data     string `json:"data"`     // CAA
}

// recordRequest is the body of a record create. Mode is left out: the API
// then uses standard mode.
type recordRequest struct {
	Name       string           `json:"name"`
	Type       string           `json:"type"`
	TTL        uint32           `json:"ttl"`
	Value      []map[string]any `json:"value"`
	SkipLookup bool             `json:"skipLookup,omitempty"`
}

// recordUpdate is the body of a record update. A PUT is partial on the API
// side, so it always carries the complete value list.
type recordUpdate struct {
	TTL        uint32           `json:"ttl"`
	Value      []map[string]any `json:"value"`
	SkipLookup bool             `json:"skipLookup,omitempty"`
}

// apiError is a response with an HTTP status of 400 or above.
type apiError struct {
	Method     string
	Path       string
	StatusCode int
	Message    string
	// Errors are the per-field messages of a validation error, as
	// "field: message".
	Errors []string
}

func (e *apiError) Error() string {
	msg := fmt.Sprintf("CONSTELLIX: %s %s: HTTP %d: %s", e.Method, e.Path, e.StatusCode, e.Message)
	if len(e.Errors) > 0 {
		msg += " (" + strings.Join(e.Errors, "; ") + ")"
	}
	if e.StatusCode == http.StatusUnauthorized {
		msg += "; the API key and secret key were rejected (a wrong key, or a system clock that is off, are the usual causes: the token embeds the local time)"
	}
	return msg
}

func newAPIClient(baseURL, apiKey, secretKey string) *apiClient {
	return &apiClient{
		baseURL:   baseURL,
		apiKey:    apiKey,
		secretKey: secretKey,
		userAgent: "dnscontrol/" + version.Version(),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		now:   time.Now,
		sleep: time.Sleep,
	}
}

func (c *apiClient) listDomains() ([]apiDomain, error) {
	return listAll[apiDomain](c, "/domains")
}

func (c *apiClient) getDomain(id flexID) (apiDomain, error) {
	var resp itemResponse[apiDomain]
	err := c.doRequest(http.MethodGet, domainPath(id), nil, nil, &resp)
	return resp.Data, err
}

func (c *apiClient) createDomain(name string) (flexID, error) {
	var resp itemResponse[struct {
		ID flexID `json:"id"`
	}]
	if err := c.doRequest(http.MethodPost, "/domains", nil, map[string]string{"name": name}, &resp); err != nil {
		return "", err
	}
	if resp.Data.ID == "" {
		return "", fmt.Errorf("CONSTELLIX: POST /domains: the response carries no zone id")
	}
	return resp.Data.ID, nil
}

func (c *apiClient) listRecords(domainID flexID) ([]apiRecord, error) {
	return listAll[apiRecord](c, domainPath(domainID)+"/records")
}

func (c *apiClient) createRecord(domainID flexID, req recordRequest) error {
	return c.doRequest(http.MethodPost, domainPath(domainID)+"/records", nil, req, nil)
}

func (c *apiClient) updateRecord(domainID, recordID flexID, req recordUpdate) error {
	return c.doRequest(http.MethodPut, recordPath(domainID, recordID), nil, req, nil)
}

func (c *apiClient) deleteRecord(domainID, recordID flexID) error {
	return c.doRequest(http.MethodDelete, recordPath(domainID, recordID), nil, nil, nil)
}

func domainPath(id flexID) string {
	return "/domains/" + url.PathEscape(string(id))
}

func recordPath(domainID, recordID flexID) string {
	return domainPath(domainID) + "/records/" + url.PathEscape(string(recordID))
}

// listAll pages through a list endpoint until it has the number of items the
// API says there are. A page that adds nothing before then is an error rather
// than a silently incomplete list.
func listAll[T any](c *apiClient, path string) ([]T, error) {
	var all []T
	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("perPage", strconv.Itoa(pageSize))
		query.Set("page", strconv.Itoa(page))

		var resp listResponse[T]
		if err := c.doRequest(http.MethodGet, path, query, nil, &resp); err != nil {
			return nil, err
		}

		all = append(all, resp.Data...)
		if len(all) >= resp.Meta.Pagination.Total {
			return all, nil
		}
		if len(resp.Data) == 0 {
			return nil, fmt.Errorf("CONSTELLIX: GET %s: API returned %d of %d items", path, len(all), resp.Meta.Pagination.Total)
		}
	}
}

// token returns the value of the Authorization header for a request sent now.
func (c *apiClient) token() string {
	ts := strconv.FormatInt(c.now().UnixMilli(), 10)
	mac := hmac.New(sha1.New, []byte(c.secretKey))
	mac.Write([]byte(ts))
	return "Bearer " + c.apiKey + ":" + base64.StdEncoding.EncodeToString(mac.Sum(nil)) + ":" + ts
}

// doRequest performs one API call and decodes the response into result.
//
// A 429 means the request was not processed; it is sent again once the window
// has reset, POST included. Should a resent POST find its first attempt
// applied after all, the API answers that the record already exists, so the
// worst case is a failed correction, never a duplicate. Nothing else is
// retried.
func (c *apiClient) doRequest(method, path string, query url.Values, body, result any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var waited time.Duration
	for {
		if err := c.wait(&waited, method, path); err != nil {
			return err
		}

		status, header, respBody, err := c.send(method, u, payload)
		if err != nil {
			return err
		}
		pause := c.noteRateLimit(status, header)

		if status == http.StatusTooManyRequests {
			apiErr := newAPIError(method, path, status, respBody)
			if waited+pause > maxRateLimitWait {
				return fmt.Errorf("%w; rate limit exceeded, giving up after waiting %s", apiErr, waited.Round(time.Second))
			}
			printer.Printf("CONSTELLIX: rate limit reached, waiting %s\n", pause.Round(time.Second))
			continue
		}
		if status >= 400 {
			return newAPIError(method, path, status, respBody)
		}

		if result == nil {
			return nil
		}
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("CONSTELLIX: %s %s: decoding response: %w", method, path, err)
		}
		return nil
	}
}

// wait blocks until the gate is open, adding the time spent to *waited.
// Another request may extend the pause while this one sleeps, so the gate is
// checked again after every sleep.
func (c *apiClient) wait(waited *time.Duration, method, path string) error {
	for {
		c.mu.Lock()
		d := c.pauseUntil.Sub(c.now())
		c.mu.Unlock()
		if d <= 0 {
			return nil
		}
		if *waited+d > maxRateLimitWait {
			return fmt.Errorf("CONSTELLIX: %s %s: rate limit exceeded, giving up after waiting %s", method, path, waited.Round(time.Second))
		}
		c.sleep(d)
		*waited += d
	}
}

// noteRateLimit closes the gate when a response says the window is used up:
// on a 429, or when X-RateLimit-Remaining is 0. It returns how long the gate
// stays closed from now.
func (c *apiClient) noteRateLimit(status int, header http.Header) time.Duration {
	if status != http.StatusTooManyRequests && header.Get("X-RateLimit-Remaining") != "0" {
		return 0
	}

	pause := rateLimitPause
	if s, err := strconv.Atoi(header.Get("X-RateLimit-Reset")); err == nil && s >= 0 {
		pause = time.Duration(s)*time.Second + rateLimitMargin
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if until := now.Add(pause); until.After(c.pauseUntil) {
		c.pauseUntil = until
	}
	return c.pauseUntil.Sub(now)
}

// send sends one request with a fresh token and reads the whole response.
func (c *apiClient) send(method, u string, payload []byte) (int, http.Header, []byte, error) {
	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("CONSTELLIX: %w", err)
	}
	req.Header.Set("Authorization", c.token())
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("CONSTELLIX: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("CONSTELLIX: %s %s: %w", method, req.URL.Path, err)
	}
	return resp.StatusCode, resp.Header, respBody, nil
}

// newAPIError builds the error for a response with a status of 400 or above.
// The API answers {"message": "..."} and, for a validation error, adds
// "errors": {"field": "message"} where the message is sometimes a string and
// sometimes a list of strings. Anything else is reported as the beginning of
// the raw body.
func newAPIError(method, path string, status int, body []byte) *apiError {
	e := &apiError{Method: method, Path: path, StatusCode: status}

	var envelope struct {
		Message string                     `json:"message"`
		Errors  map[string]json.RawMessage `json:"errors"`
	}
	if json.Unmarshal(body, &envelope) != nil || (envelope.Message == "" && len(envelope.Errors) == 0) {
		raw := strings.TrimSpace(string(body))
		if len(raw) > maxErrorBody {
			raw = raw[:maxErrorBody] + "..."
		}
		e.Message = raw
		return e
	}

	e.Message = envelope.Message
	for _, field := range slices.Sorted(maps.Keys(envelope.Errors)) {
		raw := envelope.Errors[field]
		var single string
		var list []string
		switch {
		case json.Unmarshal(raw, &single) == nil:
			e.Errors = append(e.Errors, field+": "+single)
		case json.Unmarshal(raw, &list) == nil:
			for _, m := range list {
				e.Errors = append(e.Errors, field+": "+m)
			}
		default:
			e.Errors = append(e.Errors, field+": "+string(raw))
		}
	}
	return e
}
