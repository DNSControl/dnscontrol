package nameocean

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
)

// defaultAPIURL is the NameOcean API v1 base URL.
const defaultAPIURL = "https://nameocean.net/api/v1"

// apiError is the error body NameOcean returns: {"error": "...", "code": "..."}.
// A 400 on apply also carries "errors", one entry per invalid record.
type apiError struct {
	Status  int               `json:"-"`
	Message string            `json:"error"`
	Code    string            `json:"code"`
	Errors  []json.RawMessage `json:"errors"`
}

func (e *apiError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "nameocean: HTTP %d", e.Status)
	if e.Code != "" {
		fmt.Fprintf(&b, " (%s)", e.Code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	for _, detail := range e.Errors {
		fmt.Fprintf(&b, "\n  %s", string(detail))
	}
	if e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden {
		b.WriteString("\n(the API key needs the dns:read and dns:write scopes, and the account needs two-step login)")
	}
	return b.String()
}

// errZoneNotFound is returned when the zone does not exist in the account.
var errZoneNotFound = errors.New("zone not found")

// zoneDocument is a whole zone as GET /dns/{zone}/export returns it and
// POST /dns/{zone}/apply accepts it. Records stay raw so that records this
// provider does not manage (REDIRECT, PTR) can be sent back byte for byte.
type zoneDocument struct {
	Schema  string            `json:"$schema,omitempty"`
	Zone    string            `json:"zone"`
	Records []json.RawMessage `json:"records"`
}

type zoneListResponse struct {
	Zones []struct {
		Name string `json:"name"`
	} `json:"zones"`
}

// applyResponse is the part of a POST /dns/{zone}/apply response this provider reads.
type applyResponse struct {
	Zone    string `json:"zone"`
	Applied bool   `json:"applied"`
	Summary struct {
		Create    int `json:"create"`
		Update    int `json:"update"`
		Delete    int `json:"delete"`
		Unchanged int `json:"unchanged"`
	} `json:"summary"`
}

// maxRetries is how often a request is repeated after HTTP 429.
const maxRetries = 6

// retryAfter turns a Retry-After header (seconds) into a wait, 1s to 65s.
func retryAfter(value string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || secs < 1 {
		secs = 5
	}
	if secs > 65 {
		secs = 65
	}
	return time.Duration(secs) * time.Second
}

func (p *nameoceanProvider) do(method, path string, query url.Values, body any) (*http.Response, []byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		reader = bytes.NewReader(data)
	}
	u := p.apiURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(method, u, reader)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	var resp *http.Response
	var data []byte
	for attempt := 0; ; attempt++ {
		if attempt > 0 && reader != nil {
			// the body was consumed by the previous attempt
			if seeker, ok := reader.(io.Seeker); ok {
				if _, err := seeker.Seek(0, io.SeekStart); err != nil {
					return nil, nil, err
				}
			}
		}
		resp, err = p.client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		data, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, nil, err
		}
		// 120 requests a minute per API key: wait as long as the API says, then try again
		if resp.StatusCode != http.StatusTooManyRequests || attempt >= maxRetries {
			break
		}
		wait := retryAfter(resp.Header.Get("Retry-After"))
		printer.Debugf("NAMEOCEAN: rate limited, retrying in %s\n", wait)
		time.Sleep(wait)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		apiErr := &apiError{Status: resp.StatusCode}
		if json.Unmarshal(data, apiErr) != nil || (apiErr.Message == "" && apiErr.Code == "") {
			apiErr.Message = strings.TrimSpace(string(data))
		}
		return resp, data, apiErr
	}
	return resp, data, nil
}

// listZones returns the names of all zones in the account (GET /dns).
func (p *nameoceanProvider) listZones() ([]string, error) {
	_, data, err := p.do(http.MethodGet, "/dns", nil, nil)
	if err != nil {
		return nil, err
	}
	var r zoneListResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("nameocean: decoding zone list: %w", err)
	}
	zones := make([]string, 0, len(r.Zones))
	for _, z := range r.Zones {
		zones = append(zones, strings.ToLower(strings.TrimSuffix(z.Name, ".")))
	}
	return zones, nil
}

// createZone creates a zone (POST /dns). A zone that already exists is not an error.
func (p *nameoceanProvider) createZone(name string) error {
	_, _, err := p.do(http.MethodPost, "/dns", nil, map[string]string{"name": name})
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusConflict && apiErr.Code == "already_exists" {
		return nil
	}
	return err
}

// exportZone downloads the zone document and its serial
// (GET /dns/{zone}/export?output=json).
func (p *nameoceanProvider) exportZone(zone string) (*zoneDocument, string, error) {
	query := url.Values{"output": {"json"}}
	resp, data, err := p.do(http.MethodGet, "/dns/"+url.PathEscape(zone)+"/export", query, nil)
	if err != nil {
		var apiErr *apiError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil, "", fmt.Errorf("nameocean: %q: %w (%s)", zone, errZoneNotFound, apiErr.Message)
		}
		return nil, "", err
	}
	var doc zoneDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, "", fmt.Errorf("nameocean: decoding zone %q: %w", zone, err)
	}
	return &doc, resp.Header.Get("X-Zone-Serial"), nil
}

// applyZone makes the zone match doc exactly, in one transaction
// (POST /dns/{zone}/apply?serial=...). The server refuses with 409
// zone_changed when serial is set and the zone changed since the export.
func (p *nameoceanProvider) applyZone(zone, serial string, doc *zoneDocument) (*applyResponse, error) {
	var query url.Values
	if serial != "" {
		query = url.Values{"serial": {serial}}
	}
	_, data, err := p.do(http.MethodPost, "/dns/"+url.PathEscape(zone)+"/apply", query, doc)
	if err != nil {
		return nil, err
	}
	var r applyResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("nameocean: decoding apply response for %q: %w", zone, err)
	}
	if !r.Applied {
		return &r, fmt.Errorf("nameocean: zone %q was not applied: %s", zone, strings.TrimSpace(string(data)))
	}
	return &r, nil
}
