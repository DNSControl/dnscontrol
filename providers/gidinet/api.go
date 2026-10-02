package gidinet

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

const (
	// DNS API endpoint.
	dnsAPIURL     = "https://api.quickservicebox.com/API/Beta/DNSAPI.asmx"
	dnsSoapAction = "https://api.quickservicebox.com/DNS/DNSAPI/"

	// Core API endpoint (for domain listing).
	coreAPIURL     = "https://api.quickservicebox.com/API/Beta/CoreAPI.asmx"
	coreSoapAction = "http://api.quickservicebox.com/API/Beta/CoreAPI/"
)

// gidinetProvider holds the API credentials and HTTP client.
type gidinetProvider struct {
	username    string
	passwordB64 string
	client      *http.Client
}

func (c *gidinetProvider) initializeClient(username, password string) {
	*c = gidinetProvider{
		username:    username,
		passwordB64: base64.StdEncoding.EncodeToString([]byte(password)),
		client:      &http.Client{},
	}
}

// buildSOAPRequest creates a SOAP envelope with the given body content.
func buildSOAPRequest(body any) ([]byte, error) {
	// Build XML manually to handle namespaces properly
	bodyXML, err := xml.MarshalIndent(body, "    ", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP body: %w", err)
	}

	soapEnvelope := `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
` + string(bodyXML) + `
  </soap:Body>
</soap:Envelope>`

	return []byte(soapEnvelope), nil
}

// doSOAPRequest sends a SOAP request to the DNS API and returns the response body.
func (c *gidinetProvider) doSOAPRequest(action string, requestBody any) ([]byte, error) {
	return c.doSOAPRequestToURL(dnsAPIURL, dnsSoapAction+action, requestBody)
}

// doCoreAPIRequest sends a SOAP request to the Core API and returns the response body.
func (c *gidinetProvider) doCoreAPIRequest(action string, requestBody any) ([]byte, error) {
	return c.doSOAPRequestToURL(coreAPIURL, coreSoapAction+action, requestBody)
}

// doSOAPRequestToURL sends a SOAP request to the specified URL and returns the response body.
func (c *gidinetProvider) doSOAPRequestToURL(url, soapAction string, requestBody any) ([]byte, error) {
	soapData, err := buildSOAPRequest(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(soapData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", soapAction)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// parseSOAPResponse extracts the response from a SOAP envelope.
func parseSOAPResponse(data []byte, response any) error {
	// Find the Body content and unmarshal it
	// We need to handle the SOAP envelope wrapper
	type envelope struct {
		Body struct {
			InnerXML []byte `xml:",innerxml"`
		} `xml:"Body"`
	}

	var env envelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("failed to parse SOAP envelope: %w", err)
	}

	if err := xml.Unmarshal(env.Body.InnerXML, response); err != nil {
		return fmt.Errorf("failed to parse SOAP response body: %w", err)
	}

	return nil
}

// recordGetList fetches all DNS records for a domain.
func (c *gidinetProvider) recordGetList(domainName string) ([]*DNSRecordListItem, error) {
	request := &RecordGetListRequest{
		AccountUsername:    c.username,
		AccountPasswordB64: c.passwordB64,
		DomainName:         domainName,
	}

	responseData, err := c.doSOAPRequest("recordGetList", request)
	if err != nil {
		return nil, err
	}

	var response RecordGetListResponse
	if err := parseSOAPResponse(responseData, &response); err != nil {
		return nil, err
	}

	if response.ResultCode != ResultCodeSuccess {
		return nil, fmt.Errorf("gidinet API error (code %d): %s", response.ResultCode, response.ResultText)
	}

	return response.ResultItems, nil
}

// recordAdd creates a new DNS record.
func (c *gidinetProvider) recordAdd(record *DNSRecord) error {
	request := &RecordAddRequest{
		AccountUsername:    c.username,
		AccountPasswordB64: c.passwordB64,
		Record:             record,
	}

	responseData, err := c.doSOAPRequest("recordAdd", request)
	if err != nil {
		return err
	}

	var response RecordAddResponse
	if err := parseSOAPResponse(responseData, &response); err != nil {
		return err
	}

	if response.ResultCode != ResultCodeSuccess {
		return fmt.Errorf("gidinet API recordAdd error (code %d): %s", response.ResultCode, response.ResultText)
	}

	return nil
}

// recordUpdate modifies an existing DNS record.
func (c *gidinetProvider) recordUpdate(oldRecord, newRecord *DNSRecord) error {
	request := &RecordUpdateRequest{
		AccountUsername:    c.username,
		AccountPasswordB64: c.passwordB64,
		OldRecord:          oldRecord,
		NewRecord:          newRecord,
	}

	responseData, err := c.doSOAPRequest("recordUpdate", request)
	if err != nil {
		return err
	}

	var response RecordUpdateResponse
	if err := parseSOAPResponse(responseData, &response); err != nil {
		return err
	}

	if response.ResultCode != ResultCodeSuccess {
		return fmt.Errorf("gidinet API recordUpdate error (code %d): %s", response.ResultCode, response.ResultText)
	}

	return nil
}

// recordDelete removes a DNS record.
func (c *gidinetProvider) recordDelete(record *DNSRecord) error {
	request := &RecordDeleteRequest{
		AccountUsername:    c.username,
		AccountPasswordB64: c.passwordB64,
		Record:             record,
	}

	responseData, err := c.doSOAPRequest("recordDelete", request)
	if err != nil {
		return err
	}

	var response RecordDeleteResponse
	if err := parseSOAPResponse(responseData, &response); err != nil {
		return err
	}

	if response.ResultCode != ResultCodeSuccess {
		return fmt.Errorf("gidinet API recordDelete error (code %d): %s", response.ResultCode, response.ResultText)
	}

	return nil
}

// domainGetList fetches all domains from the account.
func (c *gidinetProvider) domainGetList() ([]*domainListItem, error) {
	var allDomains []*domainListItem
	pageNumber := 1
	pageSize := 200 // Maximum allowed

	for {
		request := &domainGetListRequest{
			AccountUsername:     c.username,
			AccountPasswordB64:  c.passwordB64,
			OrderFieldID:        0, // Order by name
			OrderMode:           0, // Ascending
			PageSize:            pageSize,
			PageNumber:          pageNumber,
			GroupFilter:         0,  // All domains
			DomainFilter:        "", // No filter
			NameserversFilter:   "", // No filter
			RegistrantContactID: 0,  // No filter
			TechContactID:       0,  // No filter
		}

		responseData, err := c.doCoreAPIRequest("domainGetList", request)
		if err != nil {
			return nil, err
		}

		var response domainGetListResponse
		if err := parseSOAPResponse(responseData, &response); err != nil {
			return nil, err
		}

		if response.ResultCode != ResultCodeSuccess {
			return nil, fmt.Errorf("gidinet API domainGetList error (code %d): %s", response.ResultCode, response.ResultText)
		}

		// Only include active domains with DNS service
		for _, domain := range response.ResultItems {
			if domain.StatusCode == DomainStatusActive {
				allDomains = append(allDomains, domain)
			}
		}

		// Check if we need to fetch more pages
		if pageNumber >= response.TotalPages {
			break
		}
		pageNumber++
	}

	return allDomains, nil
}

// fixTTL snaps a TTL value to the nearest allowed value.
func fixTTL(ttl uint32) uint32 {
	// If TTL is larger than the largest allowed value, return the largest
	if ttl > allowedTTLValues[len(allowedTTLValues)-1] {
		return allowedTTLValues[len(allowedTTLValues)-1]
	}

	// Find the smallest allowed value that is >= ttl using binary search
	idx, _ := slices.BinarySearch(allowedTTLValues, ttl)
	if idx < len(allowedTTLValues) {
		return allowedTTLValues[idx]
	}

	return allowedTTLValues[0]
}

// toFQDN converts a hostname to FQDN if needed.
func toFQDN(hostname, domain string) string {
	if hostname == "@" || hostname == "" {
		return domain
	}
	if strings.HasSuffix(hostname, "."+domain) {
		return hostname
	}
	if before, ok := strings.CutSuffix(hostname, "."); ok {
		return before
	}
	return hostname + "." + domain
}

// fromFQDN converts a FQDN to a relative hostname.
func fromFQDN(fqdn, domain string) string {
	if fqdn == domain || fqdn == domain+"." {
		return "@"
	}
	if before, ok := strings.CutSuffix(fqdn, "."+domain); ok {
		return before
	}
	return fqdn
}

// Maximum length for a single TXT chunk in the Gidinet API.
const maxTXTChunkLen = 250

// chunkTXT splits a long TXT value into quoted chunks for the Gidinet API.
// The API rejects single strings longer than ~250 characters, but accepts
// multiple quoted segments like: "chunk1" "chunk2" "chunk3"
// Short values (<= maxTXTChunkLen) are returned as-is without quotes.
func chunkTXT(value string) string {
	if len(value) <= maxTXTChunkLen {
		return value
	}

	var chunks []string
	for len(value) > 0 {
		end := min(maxTXTChunkLen, len(value))
		chunks = append(chunks, `"`+value[:end]+`"`)
		value = value[end:]
	}
	return strings.Join(chunks, " ")
}

// unchunkTXT parses a TXT value that may be in chunked format back to a single string.
// Handles formats like: "chunk1" "chunk2" or just: plain value.
func unchunkTXT(data string) string {
	data = strings.TrimSpace(data)

	// If it doesn't start with a quote, return as-is
	if len(data) == 0 || data[0] != '"' {
		return data
	}

	// Parse quoted strings
	var result strings.Builder
	i := 0
	for i < len(data) {
		// Skip whitespace
		for i < len(data) && (data[i] == ' ' || data[i] == '\t') {
			i++
		}
		if i >= len(data) {
			break
		}

		// Expect opening quote
		if data[i] != '"' {
			// Not a quoted string, append rest and break
			result.WriteString(data[i:])
			break
		}
		i++ // skip opening quote

		// Read until closing quote
		start := i
		for i < len(data) && data[i] != '"' {
			i++
		}
		result.WriteString(data[start:i])

		if i < len(data) {
			i++ // skip closing quote
		}
	}

	return result.String()
}

// --- Registrar API methods ---

// getNameserversForDomain fetches the current nameservers for a specific domain.
func (c *gidinetProvider) getNameserversForDomain(domainName string) ([]string, error) {
	// Fetch all domains and find the one we're looking for
	allDomains, err := c.domainGetList()
	if err != nil {
		return nil, err
	}

	for _, domain := range allDomains {
		// API returns DomainName and DomainExtension separately (e.g., "example" + "com")
		fullName := domain.DomainName + "." + domain.DomainExtension
		if strings.EqualFold(fullName, domainName) {
			if domain.Nameservers == "" {
				return nil, nil
			}
			// API returns nameservers separated by semicolons
			nsList := strings.Split(domain.Nameservers, ";")
			for i, ns := range nsList {
				nsList[i] = strings.TrimSpace(ns)
			}
			return nsList, nil
		}
	}

	return nil, fmt.Errorf("domain %s not found in account", domainName)
}

// setNameservers updates the nameservers for a domain at the registrar level.
func (c *gidinetProvider) setNameservers(domainName string, nameservers []string) error {
	// Join nameservers with comma, no spaces
	nsString := strings.Join(nameservers, ",")

	request := &domainNameServersChangeRequest{
		AccountUsername:      c.username,
		AccountPasswordB64:   c.passwordB64,
		Domain:               domainName,
		Nameservers:          nsString,
		AdditionalParameters: nil,
	}

	responseData, err := c.doCoreAPIRequest("domainNameServersChange", request)
	if err != nil {
		return err
	}

	var response domainNameServersChangeResponse
	if err := parseSOAPResponse(responseData, &response); err != nil {
		return err
	}

	if response.ResultCode != ResultCodeSuccess {
		return fmt.Errorf("gidinet API domainNameServersChange error (code %d): %s", response.ResultCode, response.ResultText)
	}

	// Check operation result
	for _, item := range response.ResultItems {
		if item.ExitCode == 2 { // Failed
			return fmt.Errorf("gidinet nameserver change failed: %s", item.AdditionalDetails)
		}
	}

	return nil
}
