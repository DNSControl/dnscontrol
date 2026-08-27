package nexdns

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	// defaultAPIURL is the base URL of the API.
	defaultAPIURL = "https://api.nexdns.tech/v1"

	// defaultTTL is the TTL the API gives an rrset that is created without one.
	defaultTTL = 3600

	// zonesPerPage is the largest page the zone list accepts.
	zonesPerPage = 100

	// defaultRetryAfter is how long to wait after a 429 that carries no
	// Retry-After header.
	defaultRetryAfter = 5 * time.Second

	// Retry-After is treated as a lower bound rather than as the answer. A
	// sliding-window limiter reports the time one token needs at the average
	// release rate, which can be a second even when nothing is released until
	// the window rolls over a minute later; honouring such a value literally
	// burns a retry budget in seconds while making no progress. Backing off
	// from it converges on the true wait without needing to know the server's
	// policy.
	initialRateLimitBackoff = 2 * time.Second
	maxRateLimitBackoff     = time.Minute

	// maxRateLimitWait bounds the total time one request may spend waiting out
	// rate limits. It is generous because a push is not transactional: giving
	// up halfway leaves the zone holding some of the intended records and none
	// of the rest, which is worse for the operator than a slow run.
	maxRateLimitWait = 3 * time.Minute
)

type apiClient struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client

	// sleep is time.Sleep, replaced in tests so that exercising the rate-limit
	// backoff does not cost the wall-clock time it is waiting out.
	sleep func(time.Duration)
}

// apiResponse is the envelope every successful response is wrapped in.
type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// apiZone is a zone as returned by GET /zones/{id}.
type apiZone struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Nameservers []string `json:"nameservers"`
}

// apiZoneListItem is a zone as returned by GET /zones. Name is the canonical
// punycode name; UnicodeName is the same name in its native script and is equal
// to Name for an ASCII zone.
type apiZoneListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	UnicodeName string `json:"unicode_name"`
}

// apiRecord is one record value as returned by GET /zones/{id}/records. Name is
// relative to the zone, "@" at the apex, and Content is the assembled rdata.
type apiRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

// recordRequest is the body of a record create or update. Content holds only the
// primary value of the record; the numeric and keyword parts of MX, SRV, CAA, DS
// and TLSA are sent alongside it.
type recordRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Content      string `json:"content"`
	TTL          int    `json:"ttl,omitempty"`
	Priority     *int   `json:"priority,omitempty"`
	Weight       *int   `json:"weight,omitempty"`
	Port         *int   `json:"port,omitempty"`
	Flags        *int   `json:"flags,omitempty"`
	Tag          string `json:"tag,omitempty"`
	KeyTag       *int   `json:"keytag,omitempty"`
	Algorithm    *int   `json:"algorithm,omitempty"`
	DigestType   *int   `json:"digest_type,omitempty"`
	Usage        *int   `json:"usage,omitempty"`
	Selector     *int   `json:"selector,omitempty"`
	MatchingType *int   `json:"matching_type,omitempty"`
}

// apiError carries the status code as well as the message, so a caller can tell
// a missing zone (a normal outcome for EnsureZoneExists) from a real failure.
type apiError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *apiError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("NEXDNS: API error: HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("NEXDNS: API error: %s (HTTP %d)", e.Message, e.StatusCode)
}

// isNotFound reports whether err is the API saying the resource does not exist.
func isNotFound(err error) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func newAPIClient(baseURL, token string) *apiClient {
	return &apiClient{
		baseURL:  baseURL,
		apiToken: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		sleep: time.Sleep,
	}
}

// getZone looks a zone up by its exact name. The search parameter matches on a
// substring, so the exact name is picked out here; an internationalized zone is
// also matched on its native-script name.
func (c *apiClient) getZone(name string) (*apiZone, error) {
	zones, err := c.searchZones(name)
	if err != nil {
		return nil, err
	}

	for _, z := range zones {
		if z.Name != name && z.UnicodeName != name {
			continue
		}

		var zone apiZone
		if err := c.doRequest(http.MethodGet, "/zones/"+z.ID, nil, &zone); err != nil {
			return nil, err
		}
		return &zone, nil
	}

	return nil, &apiError{
		StatusCode: http.StatusNotFound,
		Code:       "not_found",
		Message:    fmt.Sprintf("zone %q not found", name),
	}
}

// listZones returns every zone the API key can see.
func (c *apiClient) listZones() ([]apiZoneListItem, error) {
	return c.searchZones("")
}

// searchZones pages through GET /zones, optionally narrowed by a search term.
func (c *apiClient) searchZones(search string) ([]apiZoneListItem, error) {
	query := url.Values{}
	query.Set("per_page", strconv.Itoa(zonesPerPage))
	if search != "" {
		query.Set("search", search)
	}

	var all []apiZoneListItem
	for page := 1; ; page++ {
		query.Set("page", strconv.Itoa(page))

		var zones []apiZoneListItem
		if err := c.doRequest(http.MethodGet, "/zones?"+query.Encode(), nil, &zones); err != nil {
			return nil, err
		}

		all = append(all, zones...)
		if len(zones) < zonesPerPage {
			return all, nil
		}
	}
}

func (c *apiClient) createZone(name string) error {
	return c.doRequest(http.MethodPost, "/zones", map[string]string{"name": name}, nil)
}

func (c *apiClient) listRecords(zoneID string) ([]apiRecord, error) {
	var records []apiRecord
	if err := c.doRequest(http.MethodGet, "/zones/"+zoneID+"/records", nil, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (c *apiClient) createRecord(zoneID string, rec recordRequest) error {
	return c.doRequest(http.MethodPost, "/zones/"+zoneID+"/records", rec, nil)
}

func (c *apiClient) updateRecord(zoneID, recordID string, rec recordRequest) error {
	return c.doRequest(http.MethodPut, "/zones/"+zoneID+"/records/"+recordID, rec, nil)
}

func (c *apiClient) deleteRecord(zoneID, recordID string) error {
	return c.doRequest(http.MethodDelete, "/zones/"+zoneID+"/records/"+recordID, nil, nil)
}

// doRequest performs one API call, unwrapping the response envelope into result.
// A push sends one request per record, which can run into the account's rate
// limit, so a "429 Too Many Requests" is waited out rather than reported.
func (c *apiClient) doRequest(method, path string, body, result any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	var waited time.Duration
	backoff := initialRateLimitBackoff

	for {
		status, header, respBody, err := c.send(method, path, payload)
		if err != nil {
			return err
		}

		if status == http.StatusTooManyRequests {
			wait := max(retryAfter(header), backoff)
			if waited+wait > maxRateLimitWait {
				return parseAPIError(status, respBody)
			}

			c.sleep(wait)
			waited += wait
			backoff = min(backoff*2, maxRateLimitBackoff)

			continue
		}
		if status >= 400 {
			return parseAPIError(status, respBody)
		}
		if result == nil || status == http.StatusNoContent {
			return nil
		}

		var envelope apiResponse
		if json.Unmarshal(respBody, &envelope) == nil && envelope.Status == "success" {
			return json.Unmarshal(envelope.Data, result)
		}
		return json.Unmarshal(respBody, result)
	}
}

func (c *apiClient) send(method, path string, payload []byte) (int, http.Header, []byte, error) {
	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil, err
	}

	return resp.StatusCode, resp.Header, respBody, nil
}

// retryAfter reads the Retry-After header of a rate-limited response.
func retryAfter(header http.Header) time.Duration {
	seconds, err := strconv.Atoi(header.Get("Retry-After"))
	if err != nil || seconds < 0 {
		return defaultRetryAfter
	}
	return time.Duration(seconds) * time.Second
}

// parseAPIError unwraps the error envelope. The per-field detail is kept, so a
// rejected record reports which field the API objected to.
func parseAPIError(statusCode int, body []byte) error {
	var envelope struct {
		Error *struct {
			Code    string                     `json:"code"`
			Message string                     `json:"message"`
			Details map[string]json.RawMessage `json:"details"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Error == nil {
		return &apiError{StatusCode: statusCode, Message: string(body)}
	}

	message := envelope.Error.Message
	for field, raw := range envelope.Error.Details {
		var list []string
		if json.Unmarshal(raw, &list) == nil && len(list) > 0 {
			message = fmt.Sprintf("%s (%s: %s)", message, field, list[0])
			continue
		}

		var single string
		if json.Unmarshal(raw, &single) == nil {
			message = fmt.Sprintf("%s (%s: %s)", message, field, single)
		}
	}

	return &apiError{StatusCode: statusCode, Code: envelope.Error.Code, Message: message}
}
