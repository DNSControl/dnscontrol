package mikrotik

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type mikrotikProvider struct {
	host      string
	username  string
	password  string
	zoneHints []string // optional list of zone names from creds.json "zonehints"
}

// dnsStaticRecord represents a RouterOS /ip/dns/static entry.
type dnsStaticRecord struct {
	ID             string `json:".id,omitempty"`
	Name           string `json:"name,omitempty"`
	Type           string `json:"type,omitempty"`
	Address        string `json:"address,omitempty"`
	CName          string `json:"cname,omitempty"`
	ForwardTo      string `json:"forward-to,omitempty"`
	MxExchange     string `json:"mx-exchange,omitempty"`
	MxPreference   string `json:"mx-preference,omitempty"`
	NS             string `json:"ns,omitempty"`
	SrvTarget      string `json:"srv-target,omitempty"`
	SrvPort        string `json:"srv-port,omitempty"`
	SrvPriority    string `json:"srv-priority,omitempty"`
	SrvWeight      string `json:"srv-weight,omitempty"`
	Text           string `json:"text,omitempty"`
	TTL            string `json:"ttl,omitempty"`
	MatchSubdomain string `json:"match-subdomain"`
	Disabled       string `json:"disabled,omitempty"`
	Dynamic        string `json:"dynamic,omitempty"`
	Comment        string `json:"comment"`
	Regexp         string `json:"regexp"`
	AddressList    string `json:"address-list"`
}

const apiPath = "/rest/ip/dns/static"
const forwardersPath = "/rest/ip/dns/forwarders"

// dnsForwarder represents a RouterOS /ip/dns/forwarders entry.
type dnsForwarder struct {
	ID            string `json:".id,omitempty"`
	Name          string `json:"name,omitempty"`
	DNSServers    string `json:"dns-servers,omitempty"`
	DohServers    string `json:"doh-servers,omitempty"`
	VerifyDohCert string `json:"verify-doh-cert,omitempty"`
	Disabled      string `json:"disabled,omitempty"`
}

// getAllForwarders fetches all DNS forwarders from RouterOS.
func (p *mikrotikProvider) getAllForwarders() ([]dnsForwarder, error) {
	body, err := p.doRequest(http.MethodGet, forwardersPath, nil)
	if err != nil {
		return nil, err
	}

	var fwds []dnsForwarder
	if err := json.Unmarshal(body, &fwds); err != nil {
		return nil, fmt.Errorf("mikrotik: failed to parse forwarders: %w", err)
	}
	return fwds, nil
}

// createForwarder creates a new DNS forwarder via PUT.
func (p *mikrotikProvider) createForwarder(f *dnsForwarder) error {
	_, err := p.doRequest(http.MethodPut, forwardersPath, f)
	return err
}

// updateForwarder updates an existing DNS forwarder via PATCH.
func (p *mikrotikProvider) updateForwarder(id string, f *dnsForwarder) error {
	_, err := p.doRequest(http.MethodPatch, forwardersPath+"/"+id, f)
	return err
}

// deleteForwarder deletes a DNS forwarder by ID via DELETE.
func (p *mikrotikProvider) deleteForwarder(id string) error {
	_, err := p.doRequest(http.MethodDelete, forwardersPath+"/"+id, nil)
	return err
}

// getAllRecords fetches all static DNS records from RouterOS.
func (p *mikrotikProvider) getAllRecords() ([]dnsStaticRecord, error) {
	body, err := p.doRequest(http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}

	var records []dnsStaticRecord
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("mikrotik: failed to parse records: %w", err)
	}

	return records, nil
}

// createRecord creates a new DNS static record via PUT.
func (p *mikrotikProvider) createRecord(r *dnsStaticRecord) error {
	_, err := p.doRequest(http.MethodPut, apiPath, r)
	return err
}

// updateRecord updates an existing DNS static record via PATCH.
func (p *mikrotikProvider) updateRecord(id string, r *dnsStaticRecord) error {
	_, err := p.doRequest(http.MethodPatch, apiPath+"/"+id, r)
	return err
}

// deleteRecord deletes a DNS static record by ID via DELETE.
func (p *mikrotikProvider) deleteRecord(id string) error {
	_, err := p.doRequest(http.MethodDelete, apiPath+"/"+id, nil)
	return err
}

// doRequest executes an HTTP request against the RouterOS REST API.
func (p *mikrotikProvider) doRequest(method, path string, payload any) ([]byte, error) {
	url := p.host + path

	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("mikrotik: failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("mikrotik: failed to create request: %w", err)
	}

	req.SetBasicAuth(p.username, p.password)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mikrotik: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mikrotik: failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("mikrotik: authentication failed (401)")
	}

	if resp.StatusCode >= 400 {
		var errMsg struct {
			Detail  string `json:"detail"`
			Error   int    `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &errMsg) == nil && (errMsg.Detail != "" || errMsg.Message != "") {
			msg := errMsg.Detail
			if msg == "" {
				msg = errMsg.Message
			}
			return nil, fmt.Errorf("mikrotik: API error (%d): %s", resp.StatusCode, msg)
		}
		return nil, fmt.Errorf("mikrotik: API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
