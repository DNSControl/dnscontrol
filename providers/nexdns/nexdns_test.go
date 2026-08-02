package nexdns

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestNewNexdns(t *testing.T) {
	if _, err := newNexdns(map[string]string{}, json.RawMessage{}); err == nil {
		t.Error("expected an error when api_token is missing")
	}

	p, err := newNexdns(map[string]string{"api_token": "nxd_notarealtoken"}, json.RawMessage{})
	if err != nil {
		t.Fatalf("newNexdns() error = %v", err)
	}
	if got := p.(*nexdnsProvider).client.baseURL; got != defaultAPIURL {
		t.Errorf("baseURL = %q, want %q", got, defaultAPIURL)
	}

	p, err = newNexdns(map[string]string{"api_token": "nxd_notarealtoken", "api_url": "https://api.example.com/v1"}, json.RawMessage{})
	if err != nil {
		t.Fatalf("newNexdns() error = %v", err)
	}
	if got := p.(*nexdnsProvider).client.baseURL; got != "https://api.example.com/v1" {
		t.Errorf("baseURL = %q, want the configured one", got)
	}
}

func TestGetZoneRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/zones":
			_, _ = w.Write([]byte(`{"status":"success","data":[{"id":"z1","name":"example.com","unicode_name":"example.com"}]}`))
		case "/zones/z1":
			_, _ = w.Write([]byte(`{"status":"success","data":{"id":"z1","name":"example.com","nameservers":["ns1.example.net","ns2.example.net"]}}`))
		case "/zones/z1/records":
			_, _ = w.Write([]byte(`{"status":"success","data":[
				{"id":"r1","name":"@","type":"SOA","content":"ns1.example.net. hostmaster.example.net. 1 2 3 4 5","ttl":3600},
				{"id":"r2","name":"@","type":"NS","content":"ns1.example.net.","ttl":3600},
				{"id":"r3","name":"sub","type":"NS","content":"ns1.example.org.","ttl":3600},
				{"id":"r4","name":"www","type":"A","content":"203.0.113.10","ttl":300}
			]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	p := testProvider(server.URL)
	recs, err := p.GetZoneRecords(&models.DomainConfig{Name: "example.com"})
	if err != nil {
		t.Fatalf("GetZoneRecords() error = %v", err)
	}

	// The SOA and the apex NS are maintained by the platform and must not be
	// reported; the NS record of a delegated child must be.
	if len(recs) != 2 {
		t.Fatalf("GetZoneRecords() returned %d records, want 2: %v", len(recs), recs)
	}
	if recs[0].Type != "NS" || recs[0].GetLabel() != "sub" {
		t.Errorf("first record = %s %s, want NS sub", recs[0].Type, recs[0].GetLabel())
	}
	if recs[1].Type != "A" || recs[1].GetTargetField() != "203.0.113.10" {
		t.Errorf("second record = %s %s, want A 203.0.113.10", recs[1].Type, recs[1].GetTargetField())
	}
	if recs[1].Original.(apiRecord).ID != "r4" {
		t.Errorf("record id was not carried over: %v", recs[1].Original)
	}
}

func TestGetNameservers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/zones":
			_, _ = w.Write([]byte(`{"status":"success","data":[{"id":"z1","name":"example.com"}]}`))
		case "/zones/z1":
			_, _ = w.Write([]byte(`{"status":"success","data":{"id":"z1","name":"example.com","nameservers":["ns1.example.net","ns2.example.net"]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	nses, err := testProvider(server.URL).GetNameservers("example.com")
	if err != nil {
		t.Fatalf("GetNameservers() error = %v", err)
	}
	if len(nses) != 2 || nses[0].Name != "ns1.example.net" {
		t.Errorf("GetNameservers() = %v, want the two nameservers of the zone", nses)
	}
}

func TestListZonesPagesThroughTheAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// A full page must be followed by a request for the next one.
		if r.URL.Query().Get("page") == "1" {
			var sb strings.Builder
			sb.WriteString(`{"status":"success","data":[`)
			for i := range zonesPerPage {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(`{"id":"z1","name":"example.com"}`)
			}
			sb.WriteString(`]}`)
			_, _ = w.Write([]byte(sb.String()))
			return
		}
		_, _ = w.Write([]byte(`{"status":"success","data":[{"id":"z2","name":"example.net"}]}`))
	}))
	defer server.Close()

	zones, err := testProvider(server.URL).ListZones()
	if err != nil {
		t.Fatalf("ListZones() error = %v", err)
	}
	if len(zones) != zonesPerPage+1 {
		t.Fatalf("ListZones() returned %d zones, want %d", len(zones), zonesPerPage+1)
	}
	if zones[len(zones)-1] != "example.net" {
		t.Errorf("last zone = %q, want the one from the second page", zones[len(zones)-1])
	}
}

func TestEnsureZoneExists(t *testing.T) {
	var created []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/zones":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			created = append(created, body["name"])
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"success","data":{"id":"z9","name":"new.example.com"}}`))
		case r.URL.Path == "/zones":
			// The search matches on a substring, so a lookup of "ample.com"
			// would see this zone too.
			_, _ = w.Write([]byte(`{"status":"success","data":[{"id":"z1","name":"example.com"}]}`))
		case r.URL.Path == "/zones/z1":
			_, _ = w.Write([]byte(`{"status":"success","data":{"id":"z1","name":"example.com"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	p := testProvider(server.URL)
	if err := p.EnsureZoneExists(&models.DomainConfig{Name: "example.com"}); err != nil {
		t.Fatalf("EnsureZoneExists() error = %v", err)
	}
	if len(created) != 0 {
		t.Errorf("an existing zone was created again: %v", created)
	}

	if err := p.EnsureZoneExists(&models.DomainConfig{Name: "new.example.com"}); err != nil {
		t.Fatalf("EnsureZoneExists() error = %v", err)
	}
	if len(created) != 1 || created[0] != "new.example.com" {
		t.Errorf("created = %v, want [new.example.com]", created)
	}
}

// rateLimitedServer answers 429 for the first refusals requests, advertising
// retryAfter each time, then succeeds.
func rateLimitedServer(t *testing.T, refusals int, retryAfter string) (*apiClient, *[]time.Duration, *int) {
	t.Helper()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= refusals {
			w.Header().Set("Retry-After", retryAfter)
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"status":"error","error":{"code":"rate_limit_exceeded","message":"Too many requests."}}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"success","data":[]}`))
	}))
	t.Cleanup(server.Close)

	client := newAPIClient(server.URL, "nxd_notarealtoken")

	slept := []time.Duration{}
	client.sleep = func(d time.Duration) { slept = append(slept, d) }

	return client, &slept, &calls
}

func TestDoRequestWaitsOutARateLimit(t *testing.T) {
	client, _, calls := rateLimitedServer(t, 1, "0")

	if _, err := client.listRecords("z1"); err != nil {
		t.Fatalf("listRecords() error = %v", err)
	}
	if *calls != 2 {
		t.Errorf("the request was attempted %d times, want 2", *calls)
	}
}

// A sliding-window limiter can answer "retry after 1 second" while releasing
// nothing until the window rolls over, so a client that takes the header
// literally retries into the same refusal and gives up with the work half done.
// The header is a lower bound; the wait has to grow on its own.
func TestRateLimitBackoffGrowsBeyondTheAdvertisedRetryAfter(t *testing.T) {
	client, slept, calls := rateLimitedServer(t, 4, "1")

	if _, err := client.listRecords("z1"); err != nil {
		t.Fatalf("listRecords() error = %v", err)
	}
	if *calls != 5 {
		t.Errorf("the request was attempted %d times, want 5", *calls)
	}

	want := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	if len(*slept) != len(want) {
		t.Fatalf("waited %v, want %v", *slept, want)
	}
	for i, d := range want {
		if (*slept)[i] != d {
			t.Errorf("wait %d was %v, want %v", i+1, (*slept)[i], d)
		}
	}
}

// A longer Retry-After is still honoured: backing off must not shorten a wait
// the server asked for.
func TestRateLimitHonoursALongerRetryAfter(t *testing.T) {
	client, slept, _ := rateLimitedServer(t, 1, "45")

	if _, err := client.listRecords("z1"); err != nil {
		t.Fatalf("listRecords() error = %v", err)
	}
	if len(*slept) != 1 || (*slept)[0] != 45*time.Second {
		t.Errorf("waited %v, want [45s]", *slept)
	}
}

// Retrying cannot go on forever: once the budget is spent the caller gets the
// API's own error rather than a hang.
func TestRateLimitGivesUpOnceTheWaitBudgetIsSpent(t *testing.T) {
	client, slept, _ := rateLimitedServer(t, 1000, "1")

	_, err := client.listRecords("z1")
	if err == nil {
		t.Fatal("listRecords() succeeded, want a rate-limit error")
	}

	var apiErr *apiError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("listRecords() error = %v, want an HTTP 429 apiError", err)
	}

	var total time.Duration
	for _, d := range *slept {
		total += d
	}
	if total > maxRateLimitWait {
		t.Errorf("waited %v in total, which exceeds the %v budget", total, maxRateLimitWait)
	}
}

func TestParseAPIError(t *testing.T) {
	body := []byte(`{"status":"error","error":{"code":"validation_error","message":"Validation failed.","details":{"content":["Enter a valid IPv4 address."]}}}`)

	err := parseAPIError(http.StatusBadRequest, body)
	if err == nil {
		t.Fatal("parseAPIError() returned no error")
	}
	if !strings.Contains(err.Error(), "Enter a valid IPv4 address.") {
		t.Errorf("the field detail is missing from %q", err.Error())
	}

	// A body that is not the error envelope must still produce an error.
	if err := parseAPIError(http.StatusInternalServerError, []byte("boom")); err == nil {
		t.Error("parseAPIError() returned no error for an unrecognized body")
	}
}

func TestIsNotFound(t *testing.T) {
	if !isNotFound(&apiError{StatusCode: http.StatusNotFound}) {
		t.Error("isNotFound() = false for a 404")
	}
	if isNotFound(&apiError{StatusCode: http.StatusInternalServerError}) {
		t.Error("isNotFound() = true for a 500")
	}
	if isNotFound(errors.New("not an API error")) {
		t.Error("isNotFound() = true for an unrelated error")
	}
}

func testProvider(baseURL string) *nexdnsProvider {
	return &nexdnsProvider{
		client: newAPIClient(baseURL, "nxd_notarealtoken"),
		zones:  map[string]*apiZone{},
	}
}
