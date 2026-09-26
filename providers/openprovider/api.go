package openprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
)

const (
	defaultAPIURL = "https://api.openprovider.eu/v1"
	pageSize      = 500
)

var transientRetryBackoffs = [...]time.Duration{
	time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	15 * time.Second,
	30 * time.Second,
}

type apiError struct {
	Operation   string
	StatusCode  int
	Code        int
	Description string
}

func (e *apiError) Error() string {
	details := "Openprovider API error"
	if e.Code != 0 {
		details += fmt.Sprintf(" code %d", e.Code)
	}
	if e.Description != "" {
		details += ": " + e.Description
	}
	return fmt.Sprintf("OPENPROVIDER: %s: %s (HTTP %d)", e.Operation, details, e.StatusCode)
}

func isNotFound(err error) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusNotFound || apiErr.Code == 800 || apiErr.Code == 872)
}

func isRetryable(err error) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusBadRequest && apiErr.Code == 18002
}

type apiClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	sleep      func(time.Duration)

	tokenMu sync.Mutex
	token   string
}

func newAPIClient(baseURL, username, password string) (*apiClient, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("OPENPROVIDER: invalid api_url")
	}

	return &apiClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
		sleep: time.Sleep,
	}, nil
}

func (c *apiClient) getZone(name string) (*apiZone, error) {
	var zone apiZone
	query := url.Values{}
	query.Set("provider", "openprovider")
	path := "/dns/zones/" + url.PathEscape(name) + "?" + query.Encode()
	if err := c.doRequest(http.MethodGet, path, nil, &zone, "get DNS zone"); err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSuffix(zone.Name, "."), strings.TrimSuffix(name, ".")) {
		return nil, fmt.Errorf("OPENPROVIDER: get DNS zone: API returned unexpected zone %q", zone.Name)
	}
	return &zone, nil
}

func (c *apiClient) listZones() ([]apiZone, error) {
	var zones []apiZone
	for offset := 0; ; offset += pageSize {
		query := url.Values{}
		query.Set("limit", strconv.Itoa(pageSize))
		query.Set("offset", strconv.Itoa(offset))
		query.Set("order_by.name", "asc")
		query.Set("provider", "openprovider")

		var page zoneListResponse
		if err := c.doRequest(http.MethodGet, "/dns/zones?"+query.Encode(), nil, &page, "list DNS zones"); err != nil {
			return nil, err
		}
		zones = append(zones, page.Results...)
		if len(page.Results) == 0 || len(zones) >= page.Total || len(page.Results) < pageSize {
			return zones, nil
		}
	}
}

func (c *apiClient) createZone(request createZoneRequest) error {
	var response successResponse
	if err := c.doRequest(http.MethodPost, "/dns/zones", request, &response, "create DNS zone"); err != nil {
		return err
	}
	if !response.Success {
		return errors.New("OPENPROVIDER: create DNS zone: API reported success=false")
	}
	return nil
}

func (c *apiClient) listRecords(zone apiZone) ([]apiRecord, error) {
	var records []apiRecord
	for offset := 0; ; offset += pageSize {
		query := url.Values{}
		query.Set("limit", strconv.Itoa(pageSize))
		query.Set("offset", strconv.Itoa(offset))
		query.Set("zone_id", strconv.FormatInt(zone.ID, 10))
		query.Set("zone_provider", "openprovider")

		var page recordListResponse
		path := "/dns/zones/" + url.PathEscape(zone.Name) + "/records?" + query.Encode()
		if err := c.doRequest(http.MethodGet, path, nil, &page, "list DNS zone records"); err != nil {
			return nil, err
		}
		records = append(records, page.Results...)
		if len(page.Results) == 0 || len(records) >= page.Total || len(page.Results) < pageSize {
			return records, nil
		}
	}
}

func (c *apiClient) updateZone(zone apiZone, updates recordUpdates) error {
	request := updateZoneRequest{
		ID:      zone.ID,
		Name:    zone.Name,
		Records: updates,
	}
	path := "/dns/zones/" + url.PathEscape(zone.Name)
	var response successResponse
	if err := c.doRequest(http.MethodPut, path, request, &response, "update DNS zone records"); err != nil {
		return err
	}
	if !response.Success {
		return errors.New("OPENPROVIDER: update DNS zone records: API reported success=false")
	}
	return nil
}

func (c *apiClient) doRequest(method, path string, body, result any, operation string) error {
	payload, err := marshalPayload(body)
	if err != nil {
		return fmt.Errorf("OPENPROVIDER: %s: encode request: %w", operation, err)
	}

	unauthorizedRetried := false
	transientRetries := 0
	for {
		token, err := c.getToken()
		if err != nil {
			return err
		}

		response, responseBody, err := c.send(context.Background(), method, path, payload, token)
		if err != nil {
			return fmt.Errorf("OPENPROVIDER: %s: %w", operation, err)
		}
		status := response.StatusCode
		if status == http.StatusUnauthorized && !unauthorizedRetried {
			c.invalidateToken(token)
			unauthorizedRetried = true
			continue
		}

		err = decodeResponse(operation, status, responseBody, result)
		if !isRetryable(err) || transientRetries >= len(transientRetryBackoffs) {
			return err
		}

		backoff := transientRetryBackoffs[transientRetries]
		transientRetries++
		printer.Warnf("OPENPROVIDER: API temporarily unavailable, retrying in %v (attempt %d/%d)\n%s", backoff, transientRetries, len(transientRetryBackoffs), formatRetryDiagnostics(response, payload, responseBody))
		c.sleep(backoff)
	}
}

func marshalPayload(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	return json.Marshal(body)
}

func (c *apiClient) getToken() (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.token != "" {
		return c.token, nil
	}

	payload, err := json.Marshal(loginRequest{
		Username: c.username,
		Password: c.password,
	})
	if err != nil {
		return "", errors.New("OPENPROVIDER: authenticate: encode request")
	}

	httpResponse, responseBody, err := c.send(context.Background(), http.MethodPost, "/auth/login", payload, "")
	if err != nil {
		return "", fmt.Errorf("OPENPROVIDER: authenticate: %w", err)
	}

	var response loginResponse
	if err := decodeResponse("authenticate", httpResponse.StatusCode, responseBody, &response); err != nil {
		return "", err
	}
	if response.Token == "" {
		return "", errors.New("OPENPROVIDER: authenticate: API returned an empty token")
	}

	c.token = response.Token
	return c.token, nil
}

func (c *apiClient) invalidateToken(token string) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.token == token {
		c.token = ""
	}
}

func (c *apiClient) send(ctx context.Context, method, path string, payload []byte, token string) (*http.Response, []byte, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, nil, err
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, err
	}
	response.Body = io.NopCloser(bytes.NewReader(responseBody))
	return response, responseBody, nil
}

func formatRetryDiagnostics(response *http.Response, payload, responseBody []byte) string {
	request := response.Request.Clone(response.Request.Context())
	request.Header = response.Request.Header.Clone()
	if request.Header.Get("Authorization") != "" {
		request.Header.Set("Authorization", "<redacted>")
	}
	request.Body = io.NopCloser(bytes.NewReader(payload))

	requestDump, requestErr := httputil.DumpRequestOut(request, true)
	responseDump, responseErr := httputil.DumpResponse(response, true)
	if requestErr != nil || responseErr != nil {
		return fmt.Sprintf("OPENPROVIDER: could not dump retry exchange (request: %v, response: %v), response body: %q\n", requestErr, responseErr, string(responseBody))
	}
	return fmt.Sprintf("OPENPROVIDER: retry exchange:\n%s%s", requestDump, responseDump)
}

func decodeResponse(operation string, status int, responseBody []byte, result any) error {
	var envelope apiEnvelope
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return fmt.Errorf("OPENPROVIDER: %s: decode response (HTTP %d): %w", operation, status, err)
	}

	if status < http.StatusOK || status >= http.StatusMultipleChoices || envelope.Code != 0 {
		return &apiError{
			Operation:   operation,
			StatusCode:  status,
			Code:        envelope.Code,
			Description: envelope.Desc,
		}
	}
	if result == nil || len(envelope.Data) == 0 || bytes.Equal(envelope.Data, []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, result); err != nil {
		return fmt.Errorf("OPENPROVIDER: %s: decode response data (HTTP %d): %w", operation, status, err)
	}
	return nil
}
