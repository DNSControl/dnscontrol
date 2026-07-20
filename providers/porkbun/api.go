package porkbun

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/failsafehttp"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
)

const (
	baseURL = "https://api.porkbun.com/api/json/v3"
)

type porkbunProvider struct {
	apiKey    string
	secretKey string

	maxAttempts int
	maxDuration time.Duration
}

type requestParams map[string]any

type errorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type domainRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     string `json:"ttl"`
	Prio    string `json:"prio"`
	// Forwarding
	Subdomain   string `json:"subdomain"`
	Location    string `json:"location"`
	IncludePath string `json:"includePath"`
	Wildcard    string `json:"wildcard"`
}

type recordResponse struct {
	Records  []domainRecord `json:"records"`
	Forwards []domainRecord `json:"forwards"`
}

type domainListRecord struct {
	Domain string `json:"domain"`
}

type domainListResponse struct {
	Domains []domainListRecord `json:"domains"`
}

type nsResponse struct {
	Nameservers []string `json:"ns"`
}

func (c *porkbunProvider) post(endpoint string, params requestParams) ([]byte, error) {
	params["apikey"] = c.apiKey
	params["secretapikey"] = c.secretKey

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return []byte{}, err
	}

	retryPolicy := failsafehttp.NewRetryPolicyBuilder().
		WithMaxAttempts(c.maxAttempts).
		// Exponential backoff between 1.2 and 10 seconds.
		// We start at 1.2 to allow for 100ms of jitter. Porkbun doesn't like
		// retries faster than 1s.
		WithBackoff(1200*time.Millisecond, 10*time.Second).
		WithJitter(100 * time.Millisecond).
		OnRetryScheduled(func(f failsafe.ExecutionScheduledEvent[*http.Response]) {
			printer.Debugf("Porkbun API response code %d, waiting for %s until next attempt\n", f.LastResult().StatusCode, f.Delay)
		})

	if c.maxDuration > 0 {
		retryPolicy = retryPolicy.WithMaxDuration(c.maxDuration)
	}

	client := &http.Client{
		Transport: failsafehttp.NewRoundTripper(nil, retryPolicy.Build()),
	}
	req, _ := http.NewRequest(http.MethodPost, baseURL+endpoint, bytes.NewBuffer(paramsJSON))

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, retrypolicy.ErrExceeded) {
			// Return the underlying error rather than the wrapped error, which has too much detail
			return nil, retrypolicy.ErrExceeded
		}
		return nil, err
	}

	bodyString, _ := io.ReadAll(resp.Body)

	// Got error from API ?
	var errResp errorResponse
	err = json.Unmarshal(bodyString, &errResp)
	if err == nil {
		if errResp.Status == "ERROR" {
			return bodyString, fmt.Errorf("porkbun API error: %s URL:%s%s ", errResp.Message, req.Host, req.URL.RequestURI())
		}
	}

	return bodyString, nil
}

func (c *porkbunProvider) createRecord(domain string, rec requestParams) error {
	if _, err := c.post("/dns/create/"+domain, rec); err != nil {
		return fmt.Errorf("failed create record (porkbun): %w", err)
	}
	return nil
}

func (c *porkbunProvider) deleteRecord(domain string, recordID string) error {
	params := requestParams{}
	if _, err := c.post(fmt.Sprintf("/dns/delete/%s/%s", domain, recordID), params); err != nil {
		return fmt.Errorf("failed delete record (porkbun): %w", err)
	}
	return nil
}

func (c *porkbunProvider) modifyRecord(domain string, recordID string, rec requestParams) error {
	if _, err := c.post(fmt.Sprintf("/dns/edit/%s/%s", domain, recordID), rec); err != nil {
		return fmt.Errorf("failed update (porkbun): %w", err)
	}
	return nil
}

func (c *porkbunProvider) getRecords(domain string) ([]domainRecord, error) {
	params := requestParams{}
	bodyString, err := c.post("/dns/retrieve/"+domain, params)
	if err != nil {
		return nil, fmt.Errorf("failed fetching record list from porkbun: %w", err)
	}

	var dr recordResponse
	err = json.Unmarshal(bodyString, &dr)
	if err != nil {
		return nil, fmt.Errorf("failed parsing record list from porkbun: %w", err)
	}

	var records []domainRecord
	for _, rec := range dr.Records {
		if rec.Name == domain && rec.Type == "NS" {
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

func (c *porkbunProvider) createURLForwardingRecord(domain string, rec requestParams) error {
	if _, err := c.post("/domain/addUrlForward/"+domain, rec); err != nil {
		return fmt.Errorf("failed create url forwarding record (porkbun): %w", err)
	}
	return nil
}

func (c *porkbunProvider) deleteURLForwardingRecord(domain string, recordID string) error {
	params := requestParams{}
	if _, err := c.post(fmt.Sprintf("/domain/deleteUrlForward/%s/%s", domain, recordID), params); err != nil {
		return fmt.Errorf("failed delete url forwarding record (porkbun): %w", err)
	}
	return nil
}

func (c *porkbunProvider) modifyURLForwardingRecord(domain string, recordID string, rec requestParams) error {
	if err := c.deleteURLForwardingRecord(domain, recordID); err != nil {
		return err
	}
	if err := c.createURLForwardingRecord(domain, rec); err != nil {
		return err
	}
	return nil
}

func (c *porkbunProvider) getURLForwardingRecords(domain string) ([]domainRecord, error) {
	params := requestParams{}
	bodyString, err := c.post("/domain/getUrlForwarding/"+domain, params)
	if err != nil {
		return nil, fmt.Errorf("failed fetching url forwarding record list from porkbun: %w", err)
	}

	var dr recordResponse
	err = json.Unmarshal(bodyString, &dr)
	if err != nil {
		return nil, fmt.Errorf("failed parsing url forwarding record list from porkbun: %w", err)
	}

	return dr.Forwards, nil
}

func (c *porkbunProvider) getNameservers(domain string) ([]string, error) {
	params := requestParams{}
	bodyString, err := c.post("/domain/getNs/"+domain, params)
	if err != nil {
		return nil, fmt.Errorf("failed fetching nameserver list from porkbun: %w", err)
	}

	var ns nsResponse
	err = json.Unmarshal(bodyString, &ns)
	if err != nil {
		return nil, fmt.Errorf("failed parsing nameserver list from porkbun: %w", err)
	}

	sort.Strings(ns.Nameservers)

	var nameservers []string
	for _, nameserver := range ns.Nameservers {
		// Remove the trailing dot only if it exists.
		// This provider seems to add the trailing dot to some domains but not others.
		// The .DE domains seem to always include the dot, others don't.
		nameservers = append(nameservers, strings.TrimSuffix(nameserver, "."))
	}
	return nameservers, nil
}

func (c *porkbunProvider) updateNameservers(ns []string, domain string) error {
	params := requestParams{}
	params["ns"] = ns
	if _, err := c.post("/domain/updateNs/"+domain, params); err != nil {
		return fmt.Errorf("failed NS update (porkbun): %w", err)
	}
	return nil
}

func (c *porkbunProvider) listAllDomains() ([]string, error) {
	params := requestParams{}
	bodyString, err := c.post("/domain/listAll", params)
	if err != nil {
		return nil, fmt.Errorf("failed listing all domains from porkbun: %w", err)
	}

	var dlr domainListResponse
	err = json.Unmarshal(bodyString, &dlr)
	if err != nil {
		return nil, fmt.Errorf("failed parsing domain list from porkbun: %w", err)
	}

	var domains []string
	for _, domain := range dlr.Domains {
		domains = append(domains, domain.Domain)
	}
	sort.Strings(domains)
	return domains, nil
}
