package loopia

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

/*
Loopia domain structure for a domain called "apex":

-apex
     +@SOA inaccessible via API
     +@zoneRecord * ... <-- use getZoneRecords(... domain: "apex", subdomain: "*")
     +@zoneRecord [NS1,NS2,TXT,TXT,A,AAAA,MX,NAPTR,etc] ... <-- use getZoneRecords(... domain: "apex", subdomain: "@")
     +subdomain1 <-- use getSubdomains(... domain: "apex")
                +zoneRecord ... <-- use getZoneRecords(... domain: "apex", subdomain: "subdomain1")
     +subdomain2
                +zoneRecord ... <-- use getZoneRecords(... domain: "apex", subdomain: "subdomain2")
     +subsubdomain1.subdomain3
                +zoneRecord ... <-- use getZoneRecords(... domain: "apex", subdomain: "subsubdomain1.subdomain3")

Note: wildcard '*' means "everything else not already defined"
getZoneRecords(... domain: "apex", subdomain: "@") returns only all @zoneRecords at the domain: "apex" level
getSubdomains(... domain: "apex") returns only all (sub)subdomains at the apex level

To build a complete local "existing/desired" zone of domain: "apex" requires at a minimum,
calls to getSubdomains, and getZoneRecords per subdomain.

*/

/*
Loopia available API functions (not necessarily implemented here):

    addDomain
    addSubdomain
    addZoneRecord
    getDomain
    getDomains
    getSubdomains
    getZoneRecords
    removeDomain
    removeSubdomain
    removeZoneRecord
    updateDNSServers
    updateZoneRecord

    domainIsFree
    getCreditsAmount
    getInvoice
    getUnpaidInvoices
    orderDomain
    payInvoiceUsingCredits
    transferDomain

Loopia available API return (object) types:

    account_type
    contact
    create_account_status_obj
    customer_obj
    domain_configuration
    domain_obj
    order_status
    order_status_obj
    invoice_obj
    invoice_item_obj
    record_obj
    status

*/

// DefaultBaseNOURL and others are RPC end-points.
const (
	DefaultBaseNOURL = "https://api.loopia.no/RPCSERV"
	DefaultBaseRSURL = "https://api.loopia.rs/RPCSERV"
	DefaultBaseSEURL = "https://api.loopia.se/RPCSERV"
)

// defaultNS1      and defaultNS2 are default NS records.
const (
	defaultNS1 = "ns1.loopia.se."
	defaultNS2 = "ns2.loopia.se."
)

// Section 2: Define the API client.

// APIClient is the APIClient handle used to store any client-related state.
type APIClient struct {
	APIUser            string
	APIPassword        string
	BaseURL            string
	HTTPClient         *http.Client
	ModifyNameServers  bool
	FetchNSEntries     bool
	Debug              bool
	requestRateLimiter requestRateLimiter
}

// NewClient creates a new LoopiaClient.
func NewClient(apiUser, apiPassword string, region string, modifyns bool, fetchns bool, debug bool) *APIClient {
	// DefaultBaseURL is url to the XML-RPC api.
	var DefaultBaseURL string
	switch region {
	case "no":
		DefaultBaseURL = DefaultBaseNOURL
	case "rs":
		DefaultBaseURL = DefaultBaseRSURL
	case "se":
		DefaultBaseURL = DefaultBaseSEURL
	default:
		DefaultBaseURL = DefaultBaseSEURL
	}
	return &APIClient{
		APIUser:           apiUser,
		APIPassword:       apiPassword,
		BaseURL:           DefaultBaseURL,
		HTTPClient:        &http.Client{Timeout: 10 * time.Second},
		ModifyNameServers: modifyns,
		FetchNSEntries:    fetchns,
		Debug:             debug,
	}
}

// CRUD: Create, Read, Update, Delete
// Create

// CreateRecordSimulate only prints info about a record addition. Used for debugging.
func (c *APIClient) CreateRecordSimulate(domain string, subdomain string, record paramStruct) error {
	if c.Debug {
		fmt.Printf("create: domain: %s; subdomain: %s; record: %+v\n", domain, subdomain, record)
	}
	return nil
}

// CreateRecord adds a record.
func (c *APIClient) CreateRecord(domain string, subdomain string, record paramStruct) error {
	call := &methodCall{
		MethodName: "addZoneRecord",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
			paramString{Value: subdomain},
			record,
		},
	}
	resp := &responseString{}

	err := c.rpcCall(call, resp)
	if err != nil {
		return err
	}

	return checkResponse(resp.Value)
}

// CRUD: Create, Read, Update, Delete
// Read

// getDomains lists all domains.
func (c *APIClient) getDomains() ([]domainObject, error) {
	call := &methodCall{
		MethodName: "getDomains",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
		},
	}
	// domainObjectsResponse is basically a zoneRecordsResponse
	resp := &domainObjectsResponse{}

	err := c.rpcCall(call, resp)

	return resp.Domains, err
}

// getDomainRecords gets all records for a subdomain.
func (c *APIClient) getDomainRecords(domain string, subdomain string) ([]zoneRecord, error) {
	call := &methodCall{
		MethodName: "getZoneRecords",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
			paramString{Value: subdomain},
		},
	}

	resp := &zoneRecordsResponse{}

	err := c.rpcCall(call, resp)

	return resp.ZoneRecords, err
}

// GetSubDomains gets all the subdomains within a domain, no records.
func (c *APIClient) GetSubDomains(domain string) ([]string, error) {
	call := &methodCall{
		MethodName: "getSubdomains",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
		},
	}

	resp := &subDomainsResponse{}

	err := c.rpcCall(call, resp)

	return resp.Params, err
}

// GetDomainNS gets all NS records for a subdomain, in this case, the apex "@".
func (c *APIClient) GetDomainNS(domain string) ([]string, error) {
	if c.ModifyNameServers {
		return nil, nil
	}

	if c.FetchNSEntries {
		return []string{defaultNS1, defaultNS2}, nil
	}

	// fetch from the domain - an extra API call.
	call := &methodCall{
		MethodName: "getZoneRecords",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
			paramString{Value: "@"},
		},
	}

	resp := &zoneRecordsResponse{}
	apexNSRecords := []string{}
	err := c.rpcCall(call, resp)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		fmt.Printf("DEBUG: getZoneRecords(@) START\n")
	}
	for i, rec := range resp.ZoneRecords {
		ns := rec.GetZR()
		if ns.Type == "NS" {
			apexNSRecords = append(apexNSRecords, ns.Rdata)
			if c.Debug {
				fmt.Printf("DEBUG: HERE %d: %v\n", i, ns)
			}
		}
	}
	return apexNSRecords, err
}

// CRUD: Create, Read, Update, Delete
// Update

// UpdateRecordSimulate only prints info about a record update. Used for debugging.
func (c *APIClient) UpdateRecordSimulate(domain string, subdomain string, rec paramStruct) error {
	fmt.Printf("got update: domain: %s; subdomain: %s; record: %v\n", domain, subdomain, rec)
	return nil
}

// UpdateRecord updates a record.
func (c *APIClient) UpdateRecord(domain string, subdomain string, rec paramStruct) error {
	call := &methodCall{
		MethodName: "updateZoneRecord",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
			paramString{Value: subdomain},
			rec,
			// // alternatively:
			// paramStruct{
			// 	StructMembers: []structMember{
			// 		structMemberString{Name: "type", Value: rtype},
			// 		structMemberInt{Name: "ttl", Value: ttl},
			// 		structMemberInt{Name: "priority", Value: prio},
			// 		structMemberString{Name: "rdata", Value: value},
			// 		structMemberInt{Name: "record_id", Value: id},
			// 	},
			// },
		},
	}
	resp := &responseString{}

	err := c.rpcCall(call, resp)
	if err != nil {
		return err
	}

	return checkResponse(resp.Value)
}

// CRUD: Create, Read, Update, Delete
// Delete

// DeleteRecordSimulate only prints info about a record deletion. Used for debugging.
func (c *APIClient) DeleteRecordSimulate(domain string, subdomain string, recordID uint32) error {
	fmt.Printf("delete: domain: %s; subdomain: %s; recordID: %d\n", domain, subdomain, recordID)
	return nil
}

// DeleteRecord deletes a record.
func (c *APIClient) DeleteRecord(domain string, subdomain string, recordID uint32) error {
	call := &methodCall{
		MethodName: "removeZoneRecord",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
			paramString{Value: subdomain},
			paramInt{Value: recordID},
		},
	}
	resp := &responseString{}

	err := c.rpcCall(call, resp)
	if err != nil {
		return err
	}

	return checkResponse(resp.Value)
}

// DeleteSubdomain deletes a sub-domain and its child records.
func (c *APIClient) DeleteSubdomain(domain, subdomain string) error {
	call := &methodCall{
		MethodName: "removeSubdomain",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
			paramString{Value: subdomain},
		},
	}
	resp := &responseString{}

	err := c.rpcCall(call, resp)
	if err != nil {
		return err
	}

	return checkResponse(resp.Value)
}

// rpcCall makes an XML-RPC call to Loopia's RPC endpoint
// by marshaling the data given in the call argument to XML and sending that via HTTP Post to Loopia.
// The response is then unmarshalled into the resp argument.
func (c *APIClient) rpcCall(call *methodCall, resp response) error {
	callBody, err := xml.MarshalIndent(call, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling the API request XML callBody: %w", err)
	}

	callBody = append([]byte(`<?xml version="1.0"?>`+"\n"), callBody...)

	// This is insecure.  Removing it until a way to sanitize callBody can be found.  --tlim 2025-08-18
	//if c.Debug {
	//	fmt.Print(string(callBody))
	//	fmt.Printf("\n")
	//}

	respBody, err := c.httpPost(c.BaseURL, "text/xml", bytes.NewReader(callBody))
	if err != nil {
		return err
	}

	if c.Debug {
		fmt.Print(string(respBody))
		fmt.Printf("\n")
	}

	err = xml.Unmarshal(respBody, resp)
	if err != nil {
		return fmt.Errorf("error unmarshalling the API response XML body: %w", err)
	}

	// yes - loopia are stoopid - the 429 error code comes from the DB behind the http proxy
	c.requestRateLimiter.handleXMLResponse(resp)
	if resp.faultCode() == 429 {
		fmt.Printf("XMLresp: %+v\n", resp)
		c.requestRateLimiter.handleRateLimitedRequest()
	} else if resp.faultCode() != 0 {
		return rpcError{
			faultCode:   resp.faultCode(),
			faultString: strings.TrimSpace(resp.faultString()),
		}
	}

	return nil
}

func (c *APIClient) httpPost(url string, bodyType string, body io.Reader) ([]byte, error) {
	c.requestRateLimiter.beforeRequest()
	resp, err := c.HTTPClient.Post(url, bodyType, body)
	c.requestRateLimiter.afterRequest()

	if err != nil {
		return nil, fmt.Errorf("HTTP Post Error: %w", err)
	}

	cleanupResponseBody := func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Printf("failed closing response body: %q\n", err)
		}
	}

	c.requestRateLimiter.handleResponse(*resp)
	// retry the request when rate-limited
	if resp.StatusCode == http.StatusTooManyRequests {
		c.requestRateLimiter.handleRateLimitedRequest()
		cleanupResponseBody()
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Post Error: %d", resp.StatusCode)
	}

	defer cleanupResponseBody()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTTP Post Error: %w", err)
	}

	return b, nil
}

func checkResponse(value string) error {
	switch v := strings.TrimSpace(value); v {
	case "OK":
		return nil
	case "AUTH_ERROR":
		return errors.New("authentication error")
	default:
		return fmt.Errorf("unknown error: %q", v)
	}
}

// Rate limiting taken from Hetzner implementation. v nice.
func getHomogenousDelay(headers http.Header, quotaName string) (time.Duration, error) {
	// Loopia, to my knowledge, are useless, and do not include such headers.
	// In the event that they one day do, use this.
	quota, err := parseHeaderAsInt(headers, "X-Ratelimit-Limit-"+cases.Title(language.Und, cases.NoLower).String((quotaName)))
	if err != nil {
		return 0, err
	}

	var unit time.Duration
	switch quotaName {
	case "hour":
		unit = time.Hour
	case "minute":
		unit = time.Minute
	case "second":
		unit = time.Second
	}

	delay := time.Duration(int64(unit) / quota)
	return delay, nil
}

func getRetryAfterDelay(header http.Header) (time.Duration, error) {
	retryAfter, err := parseHeaderAsInt(header, "Retry-After")
	if err != nil {
		return 0, err
	}
	delay := time.Duration(retryAfter * int64(time.Second))
	return delay, nil
}

func parseHeaderAsInt(headers http.Header, headerName string) (int64, error) {
	value, ok := headers[headerName]
	if !ok {
		return 0, fmt.Errorf("header %q is missing", headerName)
	}
	return strconv.ParseInt(value[0], 10, 0)
}

type requestRateLimiter struct {
	delay        time.Duration
	lastRequest  time.Time
	rateLimitPer string
}

func (requestRateLimiter *requestRateLimiter) afterRequest() {
	requestRateLimiter.lastRequest = time.Now()
}

func (requestRateLimiter *requestRateLimiter) beforeRequest() {
	if requestRateLimiter.delay == 0 {
		return
	}
	time.Sleep(time.Until(requestRateLimiter.lastRequest.Add(requestRateLimiter.delay)))
}

func (requestRateLimiter *requestRateLimiter) setDefaultDelay() {
	// default to a rate-limit of 1 req/s -- subsequent responses should update it.
	requestRateLimiter.delay = time.Second
}

func (requestRateLimiter *requestRateLimiter) setRateLimitPer(quota string) error {
	quotaNormalized := strings.ToLower(quota)
	switch quotaNormalized {
	case "hour", "minute", "second":
		requestRateLimiter.rateLimitPer = quotaNormalized
	case "":
		requestRateLimiter.rateLimitPer = "second"
	default:
		return fmt.Errorf("%q is not a valid quota, expected 'Hour', 'Minute', 'Second' or unset", quota)
	}
	return nil
}

func (requestRateLimiter *requestRateLimiter) handleRateLimitedRequest() {
	message := "Rate-Limited, consider bumping the setting 'rate_limit_per': %q -> %q"
	switch requestRateLimiter.rateLimitPer {
	case "hour":
		message = "Rate-Limited, you are already using the slowest request rate. Consider contacting Loopia Support to change this."
	case "minute":
		message = fmt.Sprintf(message, "Minute", "Hour")
	case "second":
		message = fmt.Sprintf(message, "Second", "Minute")
	}
	fmt.Print(message)
}

func (requestRateLimiter *requestRateLimiter) handleResponse(resp http.Response) {
	homogenousDelay, err := getHomogenousDelay(resp.Header, requestRateLimiter.rateLimitPer)
	if err != nil {
		requestRateLimiter.setDefaultDelay()
		return
	}

	delay := homogenousDelay
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfterDelay, err := getRetryAfterDelay(resp.Header)
		if err == nil {
			delay = retryAfterDelay
		}
	}
	requestRateLimiter.delay = delay
}

func (requestRateLimiter *requestRateLimiter) handleXMLResponse(resp response) {
	requestRateLimiter.setDefaultDelay()

	if resp.faultCode() == 429 {
		requestRateLimiter.delay = 60
	}
}
