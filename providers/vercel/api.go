package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	vercelClient "github.com/vercel/terraform-provider-vercel/client"
)

// DNSRecord is a helper struct to unmarshal the JSON response.
// It embeds vercelClient.DNSRecord to reuse the upstream type,
// but adds fields to handle API inconsistencies (type vs recordType, mxPriority).
type DNSRecord struct {
	vercelClient.DNSRecord
	Type string `json:"type"`
	// Normally MXPriority would be uint16 type, but since vercelClient.DNSRecord uses int64, we'd better be consistent here
	MXPriority int64 `json:"mxPriority"`
}

// pagination represents the pagination object in Vercel API responses.
type pagination struct {
	Count int64  `json:"count"`
	Next  *int64 `json:"next"`
	Prev  *int64 `json:"prev"`
}

// listResponse represents the response from the Vercel List DNS Records API.
type listResponse struct {
	Records    []DNSRecord `json:"records"`
	Pagination pagination  `json:"pagination"`
}

// Vercel API limit is max 100.
const vercelAPIPaginationLimit = 100

// ListDNSRecords retrieves all DNS records for a domain, handling pagination.
func (c *vercelProvider) ListDNSRecords(ctx context.Context, domain string) ([]DNSRecord, error) {
	var allRecords []DNSRecord
	var nextTimestamp int64

	for {
		url := fmt.Sprintf("https://api.vercel.com/v4/domains/%s/records?limit=%d", domain, vercelAPIPaginationLimit)
		if c.teamID != "" {
			url += fmt.Sprintf("&teamId=%s", c.teamID)
		}
		if nextTimestamp != 0 {
			url += fmt.Sprintf("&until=%d", nextTimestamp)
		}

		var result listResponse
		err := c.doRequest(clientRequest{
			ctx:    ctx,
			method: http.MethodGet,
			url:    url,
		}, &result, c.listLimiter)

		if err != nil {
			return nil, fmt.Errorf("failed to list DNS records: %w", err)
		}

		for _, r := range result.Records {
			// The official SDK expects 'recordType' but the API returns 'type'.
			// We explicitly map it here to fix the discrepancy.
			r.RecordType = r.Type
			// Ensure Domain field is set (it might not be in the record object itself)
			if r.Domain == "" {
				r.Domain = domain
			}
			if r.TeamID == "" {
				r.TeamID = c.teamID
			}

			allRecords = append(allRecords, r)
		}

		if result.Pagination.Next == nil {
			break
		}
		nextTimestamp = *result.Pagination.Next
	}

	return allRecords, nil
}

// httpsRecord structure for Vercel API.
type httpsRecord struct {
	Priority int64  `json:"priority"`
	Target   string `json:"target"`
	Params   string `json:"params"`
}

// createDNSRecordRequest embeds the official SDK request but adds HTTPS support.
type createDNSRecordRequest struct {
	vercelClient.CreateDNSRecordRequest
	Value *string      `json:"value,omitempty"`
	HTTPS *httpsRecord `json:"https,omitempty"`
}

// CreateDNSRecord creates a DNS record.
func (c *vercelProvider) CreateDNSRecord(ctx context.Context, req createDNSRecordRequest) (*vercelClient.DNSRecord, error) {
	url := fmt.Sprintf("https://api.vercel.com/v4/domains/%s/records", req.Domain)
	if c.teamID != "" {
		url += "?teamId=" + c.teamID
	}

	var response struct {
		RecordID string `json:"uid"`
	}

	payloadJSON, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: http.MethodPost,
		url:    url,
		body:   string(payloadJSON),
	}, &response, c.createLimiter)
	if err != nil {
		return nil, err
	}

	return &vercelClient.DNSRecord{ID: response.RecordID}, nil
}

// updateDNSRecordRequest embeds the official SDK request but adds HTTPS support.
type updateDNSRecordRequest struct {
	vercelClient.UpdateDNSRecordRequest
	HTTPS *httpsRecord `json:"https,omitempty"`
}

// UpdateDNSRecord updates a DNS record.
func (c *vercelProvider) UpdateDNSRecord(ctx context.Context, recordID string, req updateDNSRecordRequest) (*vercelClient.DNSRecord, error) {
	url := fmt.Sprintf("https://api.vercel.com/v4/domains/records/%s", recordID)
	if c.teamID != "" {
		url += "?teamId=" + c.teamID
	}

	payloadJSON, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var result vercelClient.DNSRecord
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: http.MethodPatch,
		url:    url,
		body:   string(payloadJSON),
	}, &result, c.updateLimiter)

	return &result, err
}

// DeleteDNSRecord deletes a DNS record.
func (c *vercelProvider) DeleteDNSRecord(ctx context.Context, domain string, recordID string) error {
	url := fmt.Sprintf("https://api.vercel.com/v2/domains/%s/records/%s", domain, recordID)
	if c.teamID != "" {
		url += "?teamId=" + c.teamID
	}

	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: http.MethodDelete,
		url:    url,
	}, nil, c.deleteLimiter)
}
