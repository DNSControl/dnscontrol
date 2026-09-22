package mittwald

import (
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/mittwald/api-client-go/pkg/httpclient"
)

// The API allows a user a number of requests per window (3000 per 600
// seconds) and reports what is left on every successful response
// (X-Ratelimit-Remaining, and X-Ratelimit-Reset in seconds). Once nothing is
// left, the next request waits for the reset.
//
// Other clients of the same user share the limit, so a request can still be
// answered with 429, which says nothing about the reset. It is repeated with
// a pause that grows to at most maxRetryPause until the total wait would
// exceed one window. A request that cannot create anything twice (not a POST)
// is repeated the same way after a 500, 502, 503 or 504.
//
// The API is eventually consistent: a write answers with the ID of its event
// (ETag), and a request that sends it as If-Event-Reached waits until the
// event is processed, or answers 412 after 10 seconds without doing anything.
// Every request after a write waits for it this way, so that a read sees it
// and a write to a zone just created finds the zone, and is repeated after a
// 412. The permissions of a zone just created can still lag behind: a request
// that is not a POST is repeated up to maxLagRetries times after a 403 or 404,
// as mittwald's own client does.
const (
	firstRetryPause = 2 * time.Second
	maxRetryPause   = time.Minute
	maxRetryWait    = 11 * time.Minute
	maxLagRetries   = 3
)

// apiRunner keeps requests within the rate limit, lets reads see the
// previous writes and repeats the requests that may be repeated.
type apiRunner struct {
	inner httpclient.RequestRunner
	sleep func(time.Duration)
	now   func() time.Time

	mu           sync.Mutex
	blockedUntil time.Time // no request before this time; the limit is used up
	lastEvent    string    // ETag of the last write
}

func newAPIRunner(inner httpclient.RequestRunner) *apiRunner {
	return &apiRunner{inner: inner, sleep: time.Sleep, now: time.Now}
}

func (r *apiRunner) Do(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	if r.lastEvent != "" {
		req.Header.Set("If-Event-Reached", r.lastEvent)
	}
	r.mu.Unlock()

	var waited time.Duration
	lagRetries := 0
	for pause := firstRetryPause; ; pause = min(2*pause, maxRetryPause) {
		r.waitForReset()
		resp, err := r.inner.Do(req)
		if err != nil {
			return resp, err
		}
		lag := lagging(req, resp)
		if lag {
			lagRetries++
		}
		repeat := retryable(req, resp) || (lag && lagRetries <= maxLagRetries)
		if !repeat {
			r.observe(req, resp)
			return resp, nil
		}
		if waited+pause > maxRetryWait {
			return resp, nil
		}
		closeBody(resp)
		if req.Body != nil {
			if req.GetBody == nil {
				return nil, errors.New("cannot repeat the request: its body cannot be sent again")
			}
			if req.Body, err = req.GetBody(); err != nil {
				return nil, err
			}
		}
		printer.Warnf("MITTWALD: %s %s: %s, waiting %s\n", req.Method, req.URL.Path, resp.Status, pause)
		r.sleep(pause)
		waited += pause
	}
}

func retryable(req *http.Request, resp *http.Response) bool {
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return true
	case http.StatusPreconditionFailed:
		return req.Header.Get("If-Event-Reached") != ""
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return req.Method != http.MethodPost
	}
	return false
}

// lagging reports a 403 or 404 that may only mean that a write before it,
// such as creating the zone, has not reached every part of the API yet.
func lagging(req *http.Request, resp *http.Response) bool {
	return req.Method != http.MethodPost && req.Header.Get("If-Event-Reached") != "" &&
		(resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound)
}

// observe remembers the event of a write, and when the limit resets once a
// response says nothing is left.
func (r *apiRunner) observe(req *http.Request, resp *http.Response) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if etag := resp.Header.Get("ETag"); etag != "" && req.Method != http.MethodGet && resp.StatusCode < http.StatusBadRequest {
		r.lastEvent = etag
	}
	remaining, err1 := strconv.Atoi(resp.Header.Get("X-Ratelimit-Remaining"))
	reset, err2 := strconv.Atoi(resp.Header.Get("X-Ratelimit-Reset"))
	if err1 == nil && err2 == nil && remaining <= 0 && reset >= 0 {
		r.blockedUntil = r.now().Add(time.Duration(reset) * time.Second)
	}
}

func (r *apiRunner) waitForReset() {
	r.mu.Lock()
	wait := r.blockedUntil.Sub(r.now())
	r.mu.Unlock()
	if wait > 0 {
		printer.Warnf("MITTWALD: rate limit used up, waiting %s for the reset\n", wait.Round(time.Second))
		r.sleep(wait)
	}
}
