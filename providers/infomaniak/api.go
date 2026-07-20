package infomaniak

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const baseURL = "https://api.infomaniak.com/2"

type dnssecRecord struct {
	IsEnabled bool `json:"is_enabled"`
}

type errorRecord struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type dnsZoneResponse struct {
	Result string      `json:"result"`
	Data   dnsZone     `json:"data"`
	Error  errorRecord `json:"error"`
}

type dnsRecordsResponse struct {
	Result string      `json:"result"`
	Data   []dnsRecord `json:"data,omitempty"`
	Error  errorRecord `json:"error"`
}

type dnsRecordResponse struct {
	Result string      `json:"result"`
	Data   dnsRecord   `json:"data"`
	Error  errorRecord `json:"error"`
}

type boolResponse struct {
	Result string      `json:"result"`
	Data   bool        `json:"data,omitempty"`
	Error  errorRecord `json:"error"`
}
type dnsZone struct {
	ID          int64        `json:"id,omitempty"`
	FQDN        string       `json:"fqdn,omitempty"`
	DNSSEC      dnssecRecord `json:"dnssec"`
	Nameservers []string     `json:"nameservers,omitempty"`
}

type dnsRecord struct {
	ID        int64  `json:"id,omitempty"`
	Source    string `json:"source,omitempty"`
	Type      string `json:"type,omitempty"`
	TTL       int64  `json:"ttl,omitempty"`
	Target    string `json:"target,omitempty"`
	UpdatedAt int64  `json:"updated_at,omitempty"`
}

type dnsRecordCreate struct {
	Source string `json:"source,omitempty"`
	Type   string `json:"type,omitempty"`
	TTL    int64  `json:"ttl,omitempty"`
	Target string `json:"target,omitempty"`
}

type dnsRecordUpdate struct {
	Target string `json:"target,omitempty"`
	TTL    int64  `json:"ttl,omitempty"`
}

func apiError(action, zone string, res *http.Response, result string, apiErr errorRecord) error {
	if result == "error" {
		if apiErr.Code != "" && apiErr.Description != "" {
			return fmt.Errorf("failed to %s for zone %s: %s (%s)", action, zone, apiErr.Description, apiErr.Code)
		}
		if apiErr.Description != "" {
			return fmt.Errorf("failed to %s for zone %s: %s", action, zone, apiErr.Description)
		}
		if apiErr.Code != "" {
			return fmt.Errorf("failed to %s for zone %s: %s", action, zone, apiErr.Code)
		}
		return fmt.Errorf("failed to %s for zone %s: unknown API error (HTTP %s)", action, zone, res.Status)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("failed to %s for zone %s: unexpected HTTP status %s", action, zone, res.Status)
	}

	return nil
}

// Get zone information
// See https://developer.infomaniak.com/docs/api/get/2/zones/%7Bzone%7D
func (p *infomaniakProvider) getDNSZone(zone string) (*dnsZone, error) {
	reqURL := fmt.Sprintf("%s/zones/%s", baseURL, zone)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+p.apiToken)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	response := &dnsZoneResponse{}
	err = json.NewDecoder(res.Body).Decode(response)
	if err != nil {
		return nil, err
	}

	if err := apiError("retrieve zone details", zone, res, response.Result, response.Error); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

// Retrieve all dns record for a given zone
// See https://developer.infomaniak.com/docs/api/get/2/zones/%7Bzone%7D/records
func (p *infomaniakProvider) getDNSRecords(zone string) ([]dnsRecord, error) {
	reqURL := fmt.Sprintf("%s/zones/%s/records", baseURL, zone)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+p.apiToken)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	response := &dnsRecordsResponse{}
	err = json.NewDecoder(res.Body).Decode(response)
	if err != nil {
		return nil, err
	}

	if err := apiError("retrieve DNS records", zone, res, response.Result, response.Error); err != nil {
		return nil, err
	}

	return response.Data, nil
}

// Delete a dns record
// See https://developer.infomaniak.com/docs/api/delete/2/zones/%7Bzone%7D/records/%7Brecord%7D
func (p *infomaniakProvider) deleteDNSRecord(zone string, recordID string) error {
	reqURL := fmt.Sprintf("%s/zones/%s/records/%s", baseURL, zone, recordID)

	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+p.apiToken)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	response := &boolResponse{}
	err = json.NewDecoder(res.Body).Decode(response)
	if err != nil {
		return err
	}

	if err := apiError("delete record "+recordID, zone, res, response.Result, response.Error); err != nil {
		return err
	}

	return nil
}

// Create a dns record in a given zone
// See https://developer.infomaniak.com/docs/api/post/2/zones/%7Bzone%7D/records
func (p *infomaniakProvider) createDNSRecord(zone string, rec *dnsRecordCreate) (*dnsRecord, error) {
	reqURL := fmt.Sprintf("%s/zones/%s/records", baseURL, zone)

	data, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+p.apiToken)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	response := &dnsRecordResponse{}
	err = json.NewDecoder(res.Body).Decode(response)
	if err != nil {
		return nil, err
	}

	if err := apiError("create "+rec.Type+" record", zone, res, response.Result, response.Error); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

// Update a dns record in a given zone
// See https://developer.infomaniak.com/docs/api/put/2/zones/%7Bzone%7D/records/%7Brecord%7D
func (p *infomaniakProvider) updateDNSRecord(zone string, recordID string, rec *dnsRecordUpdate) (*dnsRecord, error) {
	reqURL := fmt.Sprintf("%s/zones/%s/records/%s", baseURL, zone, recordID)

	data, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPut, reqURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+p.apiToken)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	response := &dnsRecordResponse{}
	err = json.NewDecoder(res.Body).Decode(response)
	if err != nil {
		return nil, err
	}

	if err := apiError("update record "+recordID, zone, res, response.Result, response.Error); err != nil {
		return nil, err
	}

	return &response.Data, nil
}
