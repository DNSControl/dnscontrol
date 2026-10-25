package constellix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeClock is a clock whose sleep only moves the time forward, so that a
// rate-limit pause costs no wall-clock time.
type fakeClock struct {
	mu    sync.Mutex
	t     time.Time
	slept []time.Duration

	// onSleep, if set, runs at the start of every sleep.
	onSleep func()
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.UnixMilli(1489401874402)}
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) sleep(d time.Duration) {
	if c.onSleep != nil {
		c.onSleep()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
	c.slept = append(c.slept, d)
}

func (c *fakeClock) sleeps() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.slept...)
}

// testClient returns a client of baseURL with the keys "key" and "s" and a
// fake clock.
func testClient(baseURL string) (*apiClient, *fakeClock) {
	clock := newFakeClock()
	c := newAPIClient(baseURL, "key", "s")
	c.now, c.sleep = clock.now, clock.sleep
	return c, clock
}

func TestToken(t *testing.T) {
	c, _ := testClient("")
	// base64(HMAC-SHA1(key="s", message="1489401874402"))
	want := "Bearer key:rIfVVYpXGBDtKk88c942eZ/WIRQ=:1489401874402"
	if got := c.token(); got != want {
		t.Errorf("token() = %q, want %q", got, want)
	}
}

func TestRequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer key:rIfVVYpXGBDtKk88c942eZ/WIRQ=:1489401874402"; got != want {
			t.Errorf("Authorization = %q, want %q", got, want)
		}
		if got := r.Header.Get("User-Agent"); !strings.HasPrefix(got, "dnscontrol/") {
			t.Errorf("User-Agent = %q, want dnscontrol/...", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data": {"id": 7}}`))
	}))
	defer server.Close()

	c, _ := testClient(server.URL)
	id, err := c.createDomain("example.com")
	if err != nil || id != "7" {
		t.Fatalf("createDomain() = %q, %v; want 7", id, err)
	}
}

func TestAPIErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   []string
	}{
		{
			name:   "401",
			status: http.StatusUnauthorized,
			body:   `{"message":"Unauthorized"}`,
			want:   []string{"GET /domains", "HTTP 401", "Unauthorized", "rejected", "clock"},
		},
		{
			name:   "404",
			status: http.StatusNotFound,
			body:   `{"message":"Not Found"}`,
			want:   []string{"GET /domains", "HTTP 404", "Not Found"},
		},
		{
			name:   "422 with a string field error",
			status: http.StatusUnprocessableEntity,
			body:   `{"message":"The given data was invalid.","errors":{"general":"\"A\" record with name \"\" in GTD Region \"Default\" already exists"}}`,
			want:   []string{"HTTP 422", "The given data was invalid.", `general: "A" record with name "" in GTD Region "Default" already exists`},
		},
		{
			name:   "422 with a list field error",
			status: http.StatusUnprocessableEntity,
			body:   `{"message":"The ttl must be between 0 and 2147483647.","errors":{"ttl":["The ttl must be between 0 and 2147483647."]}}`,
			want:   []string{"HTTP 422", "ttl: The ttl must be between 0 and 2147483647."},
		},
		{
			name:   "a body that is not JSON is truncated",
			status: http.StatusBadGateway,
			body:   "<html>" + strings.Repeat("x", 1000) + "</html>",
			want:   []string{"HTTP 502", "<html>" + strings.Repeat("x", maxErrorBody-len("<html>")) + "..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			c, _ := testClient(server.URL)
			_, err := c.listDomains()
			if err == nil {
				t.Fatal("listDomains() succeeded")
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not contain %q", err, w)
				}
			}
			if calls != 1 {
				t.Errorf("the request was sent %d times, want 1", calls)
			}
		})
	}
}

// rateLimitServer answers each request with the next of responses, then with
// 200 and an empty list. It records the fake time of every request.
type rateLimitServer struct {
	*httptest.Server
	mu    sync.Mutex
	times []time.Time
	paths []string
}

type response struct {
	status    int
	remaining string
	reset     string
}

func newRateLimitServer(clock *fakeClock, responses ...response) *rateLimitServer {
	s := &rateLimitServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		n := len(s.times)
		s.times = append(s.times, clock.now())
		s.paths = append(s.paths, r.URL.Path)
		s.mu.Unlock()

		resp := response{status: http.StatusOK, remaining: "100", reset: "30"}
		if n < len(responses) {
			resp = responses[n]
		}
		if resp.remaining != "" {
			w.Header().Set("X-RateLimit-Remaining", resp.remaining)
		}
		if resp.reset != "" {
			w.Header().Set("X-RateLimit-Reset", resp.reset)
		}
		w.WriteHeader(resp.status)
		if resp.status == http.StatusTooManyRequests {
			_, _ = w.Write([]byte(`{"message":"You have made too many requests, please wait and try again later"}`))
			return
		}
		if resp.status >= 400 {
			_, _ = w.Write([]byte(`{"message":"failed"}`))
			return
		}
		_, _ = w.Write([]byte(`{"data": [], "meta": {"pagination": {"total": 0}}}`))
	}))
	return s
}

func TestRateLimit429IsResent(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			clock := newFakeClock()
			server := newRateLimitServer(clock, response{status: http.StatusTooManyRequests, remaining: "0", reset: "42"})
			defer server.Close()
			c, _ := testClient(server.URL)
			c.now, c.sleep = clock.now, clock.sleep

			var body any
			if method == http.MethodPost {
				body = map[string]string{"name": "x"}
			}
			if err := c.doRequest(method, "/domains", nil, body, nil); err != nil {
				t.Fatalf("doRequest() error = %v", err)
			}
			if got := clock.sleeps(); len(got) != 1 || got[0] != 43*time.Second {
				t.Errorf("slept %v, want [43s]", got)
			}
			if len(server.times) != 2 || server.times[1].Sub(server.times[0]) != 43*time.Second {
				t.Errorf("requests at %v, want two 43s apart", server.times)
			}
		})
	}
}

func TestRateLimitResendUsesFreshToken(t *testing.T) {
	var tokens []string
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokens = append(tokens, r.Header.Get("Authorization"))
		calls++
		if calls == 1 {
			w.Header().Set("X-RateLimit-Reset", "5")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c, _ := testClient(server.URL)
	if err := c.doRequest(http.MethodGet, "/ping", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 2 || tokens[0] == tokens[1] {
		t.Errorf("tokens = %q, want two different ones", tokens)
	}
}

// TestRateLimitSecondRequestWaitsOnTheGate checks the gate deterministically:
// the second request is made from inside the first one's sleep. Real
// goroutines are covered by TestConcurrentReads.
func TestRateLimitSecondRequestWaitsOnTheGate(t *testing.T) {
	clock := newFakeClock()
	server := newRateLimitServer(clock, response{status: http.StatusTooManyRequests, remaining: "0", reset: "42"})
	defer server.Close()
	c, _ := testClient(server.URL)
	c.now, c.sleep = clock.now, clock.sleep

	// While the first request sleeps after its 429, a second request is
	// made. It must wait for the same reset instead of being sent.
	var second error
	clock.onSleep = func() {
		clock.onSleep = nil
		second = c.doRequest(http.MethodGet, "/second", nil, nil, nil)
	}
	if err := c.doRequest(http.MethodGet, "/first", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if second != nil {
		t.Fatal(second)
	}

	want := []string{"/first", "/second", "/first"}
	if strings.Join(server.paths, " ") != strings.Join(want, " ") {
		t.Fatalf("paths = %v, want %v", server.paths, want)
	}
	if d := server.times[1].Sub(server.times[0]); d < 43*time.Second {
		t.Errorf("the second request was sent %v after the 429, want at least 43s", d)
	}
}

func TestRateLimitRemainingZeroPausesTheNextRequest(t *testing.T) {
	clock := newFakeClock()
	server := newRateLimitServer(clock, response{status: http.StatusOK, remaining: "0", reset: "10"})
	defer server.Close()
	c, _ := testClient(server.URL)
	c.now, c.sleep = clock.now, clock.sleep

	if err := c.doRequest(http.MethodGet, "/a", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := clock.sleeps(); len(got) != 0 {
		t.Fatalf("the request that emptied the window slept %v", got)
	}
	if err := c.doRequest(http.MethodGet, "/b", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := clock.sleeps(); len(got) != 1 || got[0] != 11*time.Second {
		t.Errorf("slept %v, want [11s]", got)
	}
}

func TestRateLimitWithoutResetHeader(t *testing.T) {
	clock := newFakeClock()
	server := newRateLimitServer(clock, response{status: http.StatusTooManyRequests})
	defer server.Close()
	c, _ := testClient(server.URL)
	c.now, c.sleep = clock.now, clock.sleep

	if err := c.doRequest(http.MethodGet, "/a", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := clock.sleeps(); len(got) != 1 || got[0] != 60*time.Second {
		t.Errorf("slept %v, want [1m0s]", got)
	}
}

func TestRateLimitGivesUp(t *testing.T) {
	clock := newFakeClock()
	var responses []response
	for range 10 {
		responses = append(responses, response{status: http.StatusTooManyRequests, remaining: "0", reset: "59"})
	}
	server := newRateLimitServer(clock, responses...)
	defer server.Close()
	c, _ := testClient(server.URL)
	c.now, c.sleep = clock.now, clock.sleep

	err := c.doRequest(http.MethodGet, "/a", nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "rate limit exceeded") || !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("doRequest() error = %v, want the 429 and rate limit exceeded", err)
	}
	var total time.Duration
	for _, d := range clock.sleeps() {
		total += d
	}
	if total > maxRateLimitWait {
		t.Errorf("waited %v in total, more than %v", total, maxRateLimitWait)
	}
	if len(server.times) != 6 {
		t.Errorf("sent %d requests, want 6 (five waits of 60s fit into 5m)", len(server.times))
	}
}

func TestOtherErrorsAreNotRetried(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			clock := newFakeClock()
			server := newRateLimitServer(clock, response{status: status})
			defer server.Close()
			c, _ := testClient(server.URL)
			c.now, c.sleep = clock.now, clock.sleep

			if err := c.doRequest(http.MethodPost, "/a", nil, map[string]string{}, nil); err == nil {
				t.Fatal("doRequest() succeeded")
			}
			if len(server.times) != 1 || len(clock.sleeps()) != 0 {
				t.Errorf("sent %d requests and slept %v, want 1 and none", len(server.times), clock.sleeps())
			}
		})
	}
}

// pagedServer answers GET /domains with one page per request, in order, and
// checks the page parameters.
func pagedServer(t *testing.T, total int, pages ...[]string) *httptest.Server {
	t.Helper()
	call := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("perPage"), "100"; got != want {
			t.Errorf("perPage = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("page"), fmt.Sprint(call+1); got != want {
			t.Errorf("page = %q, want %q", got, want)
		}
		var names []string
		if call < len(pages) {
			names = pages[call]
		}
		call++

		data := make([]map[string]any, 0, len(names))
		for i, n := range names {
			data = append(data, map[string]any{"id": 100*call + i, "name": n})
		}
		resp, _ := json.Marshal(map[string]any{"data": data, "meta": map[string]any{"pagination": map[string]any{"total": total}}})
		_, _ = w.Write(resp)
	}))
}

func TestListAllPagination(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		pages   [][]string
		want    []string
		wantErr string
	}{
		{
			name:  "empty",
			total: 0,
			pages: [][]string{{}},
			want:  []string{},
		},
		{
			name:  "two pages are merged",
			total: 3,
			pages: [][]string{{"a.example", "b.example"}, {"c.example"}},
			want:  []string{"a.example", "b.example", "c.example"},
		},
		{
			name:    "a premature empty page is an error",
			total:   3,
			pages:   [][]string{{"a.example", "b.example"}, {}},
			wantErr: "API returned 2 of 3 items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := pagedServer(t, tt.total, tt.pages...)
			defer server.Close()
			c, _ := testClient(server.URL)

			domains, err := c.listDomains()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("listDomains() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, d := range domains {
				got = append(got, d.Name)
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("listDomains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlexID(t *testing.T) {
	for _, in := range []string{`{"id": 1333315}`, `{"id": "1333315"}`} {
		var v struct {
			ID flexID `json:"id"`
		}
		if err := json.Unmarshal([]byte(in), &v); err != nil || v.ID != "1333315" {
			t.Errorf("Unmarshal(%s) = %q, %v; want 1333315", in, v.ID, err)
		}
	}
	var v struct {
		ID flexID `json:"id"`
	}
	if err := json.Unmarshal([]byte(`{"id": true}`), &v); err == nil {
		t.Error("Unmarshal of a boolean id succeeded")
	}
}
