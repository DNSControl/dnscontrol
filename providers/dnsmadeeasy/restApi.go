package dnsmadeeasy

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

const (
	baseURLV2_0             = "https://api.dnsmadeeasy.com/V2.0/"
	sandboxBaseURLV2_0      = "https://api.sandbox.dnsmadeeasy.com/V2.0/"
	requestDateHeaderLayout = "Mon, 2 Jan 2006 15:04:05 MST"
	initialBackoff          = time.Second * 10 // First backoff delay duration
	maxBackoff              = time.Minute * 3  // Maximum backoff delay
)

type dnsMadeEasyRestAPI struct {
	baseURL    string
	httpClient *http.Client

	apiKey    string
	secretKey string

	dumpHTTPRequest  bool
	dumpHTTPResponse bool
}

type apiErrorResponse struct {
	Error []string `json:"error"`
}

type apiRequest struct {
	method   string
	endpoint string
	data     []byte
}

func (restApi *dnsMadeEasyRestAPI) singleDomainGet(domainID int) (*singleDomainResponse, error) {
	req := &apiRequest{
		method:   "GET",
		endpoint: fmt.Sprintf("dns/managed/%d", domainID),
	}

	res := &singleDomainResponse{}
	_, err := restApi.sendRequest(req, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (restApi *dnsMadeEasyRestAPI) multiDomainGet() (*multiDomainResponse, error) {
	req := &apiRequest{
		method:   "GET",
		endpoint: "dns/managed/",
	}

	res := &multiDomainResponse{}
	_, err := restApi.sendRequest(req, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (restApi *dnsMadeEasyRestAPI) recordGet(domainID int) (*recordResponse, error) {
	req := &apiRequest{
		method:   "GET",
		endpoint: fmt.Sprintf("dns/managed/%d/records", domainID),
	}

	res := &recordResponse{}
	_, err := restApi.sendRequest(req, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (restApi *dnsMadeEasyRestAPI) singleDomainCreate(data singleDomainRequestData) (*singleDomainResponse, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req := &apiRequest{
		method:   "POST",
		endpoint: "dns/managed/",
		data:     jsonData,
	}

	res := &singleDomainResponse{}
	_, err = restApi.sendRequest(req, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (restApi *dnsMadeEasyRestAPI) multiRecordCreate(domainID int, data []recordRequestData) (*[]recordResponseDataEntry, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req := &apiRequest{
		method:   "POST",
		endpoint: fmt.Sprintf("dns/managed/%d/records/createMulti", domainID),
		data:     jsonData,
	}

	res := &[]recordResponseDataEntry{}
	_, err = restApi.sendRequest(req, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (restApi *dnsMadeEasyRestAPI) multiRecordUpdate(domainID int, data []recordRequestData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req := &apiRequest{
		method:   "PUT",
		endpoint: fmt.Sprintf("dns/managed/%d/records/updateMulti", domainID),
		data:     jsonData,
	}

	_, err = restApi.sendRequest(req, nil)
	if err != nil {
		return err
	}

	return nil
}

func (restApi *dnsMadeEasyRestAPI) multiRecordDelete(domainID int, recordIDs []int) error {
	params := []string{}
	for i := range recordIDs {
		params = append(params, fmt.Sprintf("ids=%d", recordIDs[i]))
	}

	req := &apiRequest{
		method:   "DELETE",
		endpoint: fmt.Sprintf("dns/managed/%d/records?%s", domainID, strings.Join(params, "&")),
	}

	_, err := restApi.sendRequest(req, nil)
	if err != nil {
		return err
	}

	return nil
}

func (restApi *dnsMadeEasyRestAPI) createRequestAuthHeaders() (string, string) {
	t := time.Now()
	requestDate := t.UTC().Format(requestDateHeaderLayout)

	mac := hmac.New(sha1.New, []byte(restApi.secretKey))
	mac.Write([]byte(requestDate))
	macStr := hex.EncodeToString(mac.Sum(nil))

	return requestDate, macStr
}

func (restApi *dnsMadeEasyRestAPI) createRequest(request *apiRequest) (*http.Request, error) {
	url := restApi.baseURL + request.endpoint
	var req *http.Request
	var err error

	switch request.method {
	case "PUT", "POST":
		req, err = http.NewRequest(request.method, url, bytes.NewBuffer([]byte(request.data)))
	case "GET", "DELETE":
		req, err = http.NewRequest(request.method, url, nil)
	default:
		return nil, fmt.Errorf("unknown API request method in DNSMADEEASY REST API: %s", request.method)
	}

	if err != nil {
		return nil, err
	}

	requestDate, hmac := restApi.createRequestAuthHeaders()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-dnsme-apiKey", restApi.apiKey)
	req.Header.Set("x-dnsme-hmac", hmac)
	req.Header.Set("x-dnsme-requestDate", requestDate)

	return req, nil
}

// DNS Made Easy only allows 150 request / 5 minutes
// backoff is the amount of time to sleep if a "Rate limit exceeded" error is received
// It is increased up to maxBackoff after each use
// It is reset after successful request.
var backoff = initialBackoff

func (restApi *dnsMadeEasyRestAPI) sendRequest(request *apiRequest, response any) (int, error) {
retry:
	req, err := restApi.createRequest(request)
	if err != nil {
		return 0, err
	}

	if restApi.dumpHTTPRequest {
		dump, _ := httputil.DumpRequest(req, true)
		printer.Printf("%s", string(dump))
	}

	res, err := restApi.httpClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer res.Body.Close()

	if restApi.dumpHTTPResponse {
		dump, _ := httputil.DumpResponse(res, true)
		printer.Printf("%s", string(dump))
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		var apiErr apiErrorResponse
		err = json.NewDecoder(res.Body).Decode(&apiErr)
		if err != nil {
			return res.StatusCode, fmt.Errorf("DNSMADEEASY API unknown error, status code: %d", res.StatusCode)
		}

		if len(apiErr.Error) == 1 && apiErr.Error[0] == "Rate limit exceeded" {
			printer.Printf("pausing DNSMADEEASY due to ratelimit: %v seconds\n", backoff)

			time.Sleep(backoff)

			backoff = min(backoff+(backoff/2), maxBackoff)

			goto retry
		}

		return res.StatusCode, fmt.Errorf("DNSMADEEASY API error: %s", strings.Join(apiErr.Error, " "))
	}

	backoff = initialBackoff

	if response != nil {
		err = json.NewDecoder(res.Body).Decode(response)
		if err != nil {
			return res.StatusCode, err
		}
	}

	return res.StatusCode, nil
}
