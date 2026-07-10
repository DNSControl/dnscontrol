package gigahost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.gigahost.no/api/v0"

// httpClient has an explicit timeout so a hung connection can never block a
// run indefinitely.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// apiMeta is the "meta" object present in every Gigahost API response.
type apiMeta struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// apiEnvelope is the standard wrapper around every Gigahost API response:
// {"meta":{...},"data":...}.
type apiEnvelope struct {
	Meta apiMeta         `json:"meta"`
	Data json.RawMessage `json:"data"`
}

// zone models an entry from GET /dns/zones. All IDs arrive as JSON strings.
type zone struct {
	ZoneID      string `json:"zone_id"`
	ZoneName    string `json:"zone_name"`
	ExternalDNS string `json:"external_dns"`
}

// flexUint tolerates JSON numbers, quoted numeric strings, and null. The
// Gigahost API returns numeric fields inconsistently — for example
// record_priority comes back as the string "10" for MX records but null
// otherwise — so a plain uint cannot decode them.
type flexUint struct {
	Valid bool
	Value uint32
}

func (f *flexUint) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		return nil
	}
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return fmt.Errorf("gigahost: cannot parse numeric field %q: %w", string(b), err)
	}
	f.Value = uint32(v)
	f.Valid = true
	return nil
}

// record models an entry from GET /dns/zones/{zone_id}/records. All IDs arrive
// as JSON strings, and record_id may be non-numeric. Numeric fields may arrive
// as strings, so they use flexUint.
type record struct {
	RecordID    string   `json:"record_id,omitempty"`
	RecordName  string   `json:"record_name"`
	RecordType  string   `json:"record_type"`
	RecordValue string   `json:"record_value"`
	RecordTTL   flexUint `json:"record_ttl"`
	RecordPrio  flexUint `json:"record_priority"`
}

// recordRequest is the body sent to create (POST) and update (PUT) endpoints.
type recordRequest struct {
	RecordName  string  `json:"record_name"`
	RecordType  string  `json:"record_type"`
	RecordValue string  `json:"record_value"`
	RecordTTL   uint32  `json:"record_ttl"`
	RecordPrio  *uint16 `json:"record_priority,omitempty"`
}

// request performs an HTTP request against the Gigahost API, unwraps the
// standard envelope, surfaces meta-level errors, and decodes data into target.
func (c *gigahostProvider) request(method, path string, query url.Values, body, target any) error {
	var reqBody io.Reader
	if body != nil {
		j, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(j)
	}

	u := baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var env apiEnvelope
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("gigahost: could not decode response for %s %s (HTTP %d): %w", method, path, resp.StatusCode, err)
		}
	}

	// Prefer the envelope's meta.status, falling back to the HTTP status code.
	status := env.Meta.Status
	if status == 0 {
		status = resp.StatusCode
	}
	if status >= 400 {
		msg := env.Meta.Message
		if msg == "" {
			msg = string(raw)
		}
		return fmt.Errorf("gigahost: %s %s failed (status %d): %s", method, path, status, msg)
	}

	if target != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, target); err != nil {
			return fmt.Errorf("gigahost: could not decode data for %s %s: %w", method, path, err)
		}
	}
	return nil
}

// getAllZones returns every zone in the account (GET /dns/zones).
func (c *gigahostProvider) getAllZones() ([]zone, error) {
	var zones []zone
	if err := c.request("GET", "/dns/zones", nil, nil, &zones); err != nil {
		return nil, err
	}
	return zones, nil
}

// getRecords returns all records in a zone (GET /dns/zones/{zone_id}/records).
func (c *gigahostProvider) getRecords(zoneID string) ([]record, error) {
	var recs []record
	if err := c.request("GET", "/dns/zones/"+url.PathEscape(zoneID)+"/records", nil, nil, &recs); err != nil {
		return nil, err
	}
	return recs, nil
}

// createRecord creates a record (POST /dns/zones/{zone_id}/records).
func (c *gigahostProvider) createRecord(zoneID string, r *recordRequest) error {
	return c.request("POST", "/dns/zones/"+url.PathEscape(zoneID)+"/records", nil, r, nil)
}

// updateRecord updates a record (PUT /dns/zones/{zone_id}/records/{record_id}).
func (c *gigahostProvider) updateRecord(zoneID, recordID string, r *recordRequest) error {
	return c.request("PUT", "/dns/zones/"+url.PathEscape(zoneID)+"/records/"+url.PathEscape(recordID), nil, r, nil)
}

// deleteRecord deletes a single record. The name, type, and value query params
// are all required: name+type alone deletes EVERY record in that RRset, so value
// is needed to target one record when several share a name+type (e.g.
// round-robin A records, multiple MX/TXT). The record_id in the path is a
// content-derived hash and is not the disambiguator.
func (c *gigahostProvider) deleteRecord(zoneID, recordID, name, rtype, value string) error {
	q := url.Values{}
	q.Set("name", name)
	q.Set("type", rtype)
	q.Set("value", value)
	return c.request("DELETE", "/dns/zones/"+url.PathEscape(zoneID)+"/records/"+url.PathEscape(recordID), q, nil, nil)
}
