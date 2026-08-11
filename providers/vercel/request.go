package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	vercelClient "github.com/vercel/terraform-provider-vercel/client"
)

type clientRequest struct {
	ctx              context.Context
	method           string
	url              string
	body             string
	errorOnNoContent bool
}

func (cr *clientRequest) toHTTPRequest() (*http.Request, error) {
	r, err := http.NewRequestWithContext(
		cr.ctx,
		cr.method,
		cr.url,
		strings.NewReader(cr.body),
	)
	if err != nil {
		return nil, err
	}
	// Use a custom user agent for dnscontrol
	r.Header.Set("User-Agent", "dnscontrol https://github.com/DNSControl/dnscontrol/pull/3542")
	if cr.body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r, nil
}

// doRequest is a helper function for consistently requesting data from vercel.
// It implements rate limiting and retries.
func (c *vercelProvider) doRequest(req clientRequest, v any, rl *rateLimiter) error {
	// Use a default http client with timeout
	httpClient := &http.Client{
		Timeout: 5 * 60 * time.Second,
	}

	if rl == nil {
		panic("doRequest is expecting a rate limiter but got nil, please fire an issue and ping @SukkaW")
	}

	for {
		r, err := req.toHTTPRequest()
		if err != nil {
			return err
		}
		r.Header.Add("Authorization", "Bearer "+c.apiToken)

		rl.delayRequest()

		resp, err := httpClient.Do(r)
		if err != nil {
			return fmt.Errorf("error doing http request: %w", err)
		}

		// Handle rate limiting and retries, 429 is handled here
		retry, err := rl.handleResponse(resp)

		if err != nil {
			defer resp.Body.Close()
			return err
		}
		if retry {
			defer resp.Body.Close()
			continue
		}

		// Process response
		err = c.processResponse(resp, v, req.errorOnNoContent)
		defer resp.Body.Close()
		return err
	}
}

func (c *vercelProvider) processResponse(resp *http.Response, v any, errorOnNoContent bool) error {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode >= 300 {
		var errorResponse vercelClient.APIError
		if len(responseBody) == 0 {
			errorResponse.StatusCode = resp.StatusCode
			return errorResponse
		}

		// Try to unmarshal wrapped error first
		err = json.Unmarshal(responseBody, &struct {
			Error *vercelClient.APIError `json:"error"`
		}{
			Error: &errorResponse,
		})
		if err != nil {
			// Try to unmarshal directly if it's not wrapped in "error"
			if err2 := json.Unmarshal(responseBody, &errorResponse); err2 != nil {
				return fmt.Errorf("error unmarshaling response for status code %d: %w", resp.StatusCode, err)
			}
		}
		errorResponse.StatusCode = resp.StatusCode
		errorResponse.RawMessage = responseBody
		return errorResponse
	}

	if v == nil {
		return nil
	}

	if errorOnNoContent && resp.StatusCode == 204 {
		return vercelClient.APIError{
			StatusCode: 204,
			Code:       "no_content",
			Message:    "No content",
		}
	}

	// If we expect content but got none (and not 204), that might be an issue,
	// but json.Unmarshal will just do nothing if empty, or error.
	if len(responseBody) > 0 {
		err = json.Unmarshal(responseBody, v)
		if err != nil {
			return fmt.Errorf("error unmarshaling response %s: %w", responseBody, err)
		}
	}

	return nil
}

// rateLimiter handles Vercel's rate limits.
type rateLimiter struct {
	mu            sync.Mutex
	delay         time.Duration
	lastRequest   time.Time
	resetAt       time.Time
	defaultLimit  int64
	defaultWindow time.Duration
	remaining     int64 // Local tracking for operations without headers
}

func newRateLimiter(limit int64, window time.Duration) *rateLimiter {
	return &rateLimiter{
		defaultLimit:  limit,
		defaultWindow: window,
		remaining:     limit, // Start with full (safe) quota
		resetAt:       time.Now().Add(window),
	}
}

func (rl *rateLimiter) delayRequest() {
	rl.mu.Lock()
	// Check if we need to reset local quota
	if time.Now().After(rl.resetAt) {
		rl.remaining = rl.defaultLimit
		rl.resetAt = time.Now().Add(rl.defaultWindow)
	}

	// When not rate-limited, include network/server latency in delay.
	next := rl.lastRequest.Add(rl.delay)
	if next.After(rl.resetAt) {
		// Do not stack delays past the reset point.
		next = rl.resetAt
	}
	rl.lastRequest = next
	rl.mu.Unlock()

	wait := time.Until(next)
	if wait > 0 {
		time.Sleep(wait)
	}
}

func (rl *rateLimiter) handleResponse(resp *http.Response) (bool, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Decrement local remaining count
	if rl.remaining > 0 {
		rl.remaining--
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		printer.Printf("Rate-Limited. URL: %q, Headers: %v\n", resp.Request.URL, resp.Header)

		// Check Retry-After header first
		retryAfter, err := parseHeaderAsSeconds(resp.Header, "Retry-After", 0)
		if err == nil && retryAfter > 0 {
			rl.delay = retryAfter
			rl.lastRequest = time.Now()
			return true, nil
		}

		// Fallback to x-ratelimit-reset if Retry-After is missing/invalid
		resetAt, err := parseHeaderAsEpoch(resp.Header, "x-ratelimit-reset")
		if err == nil {
			rl.delay = time.Until(resetAt)
			if rl.delay < 0 {
				rl.delay = time.Second // Minimum delay if reset is in past
			}
			rl.lastRequest = time.Now()
			return true, nil
		}

		// Default fallback if no headers
		rl.delay = 5 * time.Second
		rl.lastRequest = time.Now()
		return true, nil
	}

	// Parse standard rate limit headers to proactively delay
	// Vercel headers: x-ratelimit-limit, x-ratelimit-remaining, x-ratelimit-reset
	// These headers are only present on Create and Update operations
	limit, err := parseHeaderAsInt(resp.Header, "x-ratelimit-limit", -1)
	if err != nil || limit == -1 {
		// Update default limit if provided
		// We don't update rl.defaultLimit permanently, but use it for calculation
		limit = rl.defaultLimit
	}

	remaining, err := parseHeaderAsInt(resp.Header, "x-ratelimit-remaining", -1)
	if err != nil || remaining == -1 {
		// Use local tracking
		remaining = rl.remaining
	} else {
		// Sync local tracking with server
		rl.remaining = remaining
	}

	resetAt, err := parseHeaderAsEpoch(resp.Header, "x-ratelimit-reset")
	if err == nil {
		rl.resetAt = resetAt
	} else {
		// Use local resetAt
		resetAt = rl.resetAt
	}

	// Apply safety factor
	safeRemaining := remaining - 2

	if safeRemaining <= 0 {
		// Quota exhausted (safely). Wait until quota resets.
		rl.delay = time.Until(resetAt)
	} else if safeRemaining > limit/2 {
		// Burst through half of the safe quota
		rl.delay = 0
	} else {
		// Spread requests evenly
		window := time.Until(resetAt)
		if window > 0 {
			rl.delay = window / time.Duration(safeRemaining+1)
		} else {
			rl.delay = 0
		}
	}

	return false, nil
}

func parseHeaderAsInt(headers http.Header, headerName string, fallback int64) (int64, error) {
	v := headers.Get(headerName)
	if v == "" {
		return fallback, nil
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback, err
	}
	return i, nil
}

func parseHeaderAsSeconds(header http.Header, headerName string, fallback time.Duration) (time.Duration, error) {
	val, err := parseHeaderAsInt(header, headerName, -1)
	if err != nil || val == -1 {
		return fallback, err
	}
	return time.Duration(val) * time.Second, nil
}

func parseHeaderAsEpoch(header http.Header, headerName string) (time.Time, error) {
	val, err := parseHeaderAsInt(header, headerName, -1)
	if err != nil || val == -1 {
		return time.Time{}, fmt.Errorf("header %s not found or invalid", headerName)
	}
	return time.Unix(val, 0), nil
}
