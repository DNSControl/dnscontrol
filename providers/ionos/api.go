package ionos

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v5/pkg/zonecache"
)

const (
	// defaultRetryAfter is how long to wait after a 429 without a Retry-After header.
	defaultRetryAfter = 5 * time.Second

	initialRateLimitBackoff = 2 * time.Second
	maxRateLimitBackoff     = time.Minute

	// maxRateLimitWait bounds the total time one request may spend on rate limits.
	maxRateLimitWait = 3 * time.Minute
)

// zone is one entry of GET /zones.
type zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// zoneDetail is the answer of GET /zones/{zoneId}.
type zoneDetail struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Records []record `json:"records"`
}

// record is a DNS record as the API returns and accepts it. Name is a FQDN
// without a trailing dot.
type record struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Prio     int    `json:"prio"`
	Disabled bool   `json:"disabled"`
}

// recordUpdate is the body of PUT /zones/{zoneId}/records/{recordId}: the API
// does not allow changing the name or the type of a record.
type recordUpdate struct {
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Prio     int    `json:"prio"`
	Disabled bool   `json:"disabled"`
}

// apiError is one entry of the error list returned by the API.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// do performs one API call. A push sends one request per record and is not
// transactional, so a "429 Too Many Requests" is waited out rather than reported:
// giving up halfway would leave the zone partly applied.
func (p *ionosProvider) do(method, path string, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
	}

	var waited time.Duration
	backoff := initialRateLimitBackoff
	for {
		status, header, data, err := p.send(method, path, payload)
		if err != nil {
			return err
		}
		if status == http.StatusTooManyRequests {
			wait := max(retryAfter(header), backoff)
			if waited+wait > maxRateLimitWait {
				return newStatusError(method, path, status, data)
			}
			p.sleep(wait)
			waited += wait
			backoff = min(backoff*2, maxRateLimitBackoff)
			continue
		}
		if status < 200 || status >= 300 {
			return newStatusError(method, path, status, data)
		}
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decoding response of %s %s: %w", method, path, err)
		}
		return nil
	}
}

// send performs one HTTP request. The body is rebuilt from payload on each call,
// because a request consumes its reader.
func (p *ionosProvider) send(method, path string, payload []byte) (int, http.Header, []byte, error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, p.baseURL+path, reader)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-API-Key", p.apiKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("reading response: %w", err)
	}
	return resp.StatusCode, resp.Header, data, nil
}

// retryAfter reads the Retry-After header of a rate-limited response, in either
// of the two forms RFC 9110 allows: a number of seconds or an HTTP-date.
func retryAfter(header http.Header) time.Duration {
	value := header.Get("Retry-After")
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return defaultRetryAfter
		}
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		return max(time.Until(date), 0)
	}
	return defaultRetryAfter
}

// statusError is a non-2xx response. It keeps the status and the error codes of
// the body so that callers can tell one failure from another.
type statusError struct {
	status int
	codes  []string
	msg    string
}

func (e *statusError) Error() string { return e.msg }

// hasCode tells whether the API answered with this status and this error code.
func (e *statusError) hasCode(status int, code string) bool {
	return e.status == status && slices.Contains(e.codes, code)
}

func newStatusError(method, path string, status int, data []byte) error {
	var details []apiError
	if err := json.Unmarshal(data, &details); err != nil || len(details) == 0 {
		return &statusError{status: status, msg: fmt.Sprintf("%s %s: HTTP %d: %s", method, path, status, data)}
	}
	parts := make([]string, 0, len(details))
	codes := make([]string, 0, len(details))
	for _, d := range details {
		parts = append(parts, d.Code+": "+d.Message)
		codes = append(codes, d.Code)
	}
	return &statusError{status: status, codes: codes, msg: fmt.Sprintf("%s %s: HTTP %d: %s", method, path, status, strings.Join(parts, "; "))}
}

// fetchAllZones is used by the zonecache.ZoneCache.
func (p *ionosProvider) fetchAllZones() (map[string]zone, error) {
	var list []zone
	if err := p.do(http.MethodGet, "/zones", nil, &list); err != nil {
		return nil, err
	}
	zones := make(map[string]zone, len(list))
	for _, z := range list {
		zones[strings.ToLower(strings.TrimSuffix(z.Name, "."))] = z
	}
	return zones, nil
}

// zoneID resolves a zone name to the identifier the other endpoints expect.
func (p *ionosProvider) zoneID(name string) (string, error) {
	z, err := p.zoneCache.GetZone(strings.ToLower(name))
	if errors.Is(err, zonecache.ErrZoneNotFound) {
		return "", fmt.Errorf("zone %s not found in the IONOS account: %w", name, err)
	}
	if err != nil {
		return "", err
	}
	return z.ID, nil
}

func (p *ionosProvider) listRecords(zoneID string) ([]record, error) {
	var detail zoneDetail
	if err := p.do(http.MethodGet, "/zones/"+url.PathEscape(zoneID), nil, &detail); err != nil {
		return nil, err
	}
	return detail.Records, nil
}

func (p *ionosProvider) createRecord(zoneID string, r record) error {
	return p.do(http.MethodPost, "/zones/"+url.PathEscape(zoneID)+"/records", []record{r}, nil)
}

func (p *ionosProvider) updateRecord(zoneID, recordID string, r record) error {
	body := recordUpdate{Content: r.Content, TTL: r.TTL, Prio: r.Prio, Disabled: r.Disabled}
	return p.do(http.MethodPut, "/zones/"+url.PathEscape(zoneID)+"/records/"+url.PathEscape(recordID), body, nil)
}

// deleteRecord treats a record that no longer exists as deleted. IONOS removes
// some records along with others: deleting the apex A record of a zone bound to
// an IONOS website also removes the apex AAAA, the www records and their
// _dep_ws_mutex TXT, so the DELETE planned for those answers 404 RECORD_NOT_FOUND.
func (p *ionosProvider) deleteRecord(zoneID, recordID string) error {
	err := p.do(http.MethodDelete, "/zones/"+url.PathEscape(zoneID)+"/records/"+url.PathEscape(recordID), nil, nil)
	var se *statusError
	if errors.As(err, &se) && se.hasCode(http.StatusNotFound, "RECORD_NOT_FOUND") {
		return nil
	}
	return err
}
