package nameocean

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func TestFixTTL(t *testing.T) {
	tests := []struct {
		rtype string
		in    uint32
		want  uint32
	}{
		{"A", 0, 300},
		{"A", 1, 300},
		{"A", 30, 300},
		{"TXT", 30, 30},
		{"TXT", 1, 30},
		{"TXT", 165, 30},  // tie between 30 and 300: the shorter
		{"TXT", 166, 300}, // closer to 300
		{"A", 300, 300},
		{"A", 600, 300},   // tie between 300 and 900: the shorter
		{"A", 601, 900},   // closer to 900
		{"MX", 1200, 900}, // closer to 900
		{"A", 1350, 900},  // tie between 900 and 1800
		{"A", 1800, 1800},
		{"A", 3600, 3600},
		{"A", 7200, 3600},
		{"A", 44999, 3600},
		{"A", 45000, 3600}, // tie between 3600 and 86400
		{"A", 45001, 86400},
		{"A", 86400, 86400},
		{"A", 604800, 86400},
	}
	for _, tt := range tests {
		if got := fixTTL(tt.rtype, tt.in); got != tt.want {
			t.Errorf("fixTTL(%s, %d) = %d, want %d", tt.rtype, tt.in, got, tt.want)
		}
	}
}

func TestParseTTL(t *testing.T) {
	tests := map[string]uint32{"": 3600, "30": 30, "5m": 300, "15m": 900, "30m": 1800, "1h": 3600, "1d": 86400}
	for in, want := range tests {
		got, err := parseTTL(in)
		if err != nil || got != want {
			t.Errorf("parseTTL(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := parseTTL("2w"); err == nil {
		t.Error("parseTTL(\"2w\") succeeded, want an error")
	}
}

func TestFormatTTLRejectsUnsnapped(t *testing.T) {
	if _, err := formatTTL(600); err == nil {
		t.Error("formatTTL(600) succeeded, want an error")
	}
}

func TestPrepareDesired(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	dc.Records = models.Records{
		dc.MustNewRecordConfig("@", 300, dnsv2.TypeNS, "ns1.nameocean.net."),
		dc.MustNewRecordConfig("@", 300, dnsv2.TypeNS, "ns1.other.example."),
		dc.MustNewRecordConfig("sub", 300, dnsv2.TypeNS, "ns1.other.example."),
		dc.MustNewRecordConfig("www", 600, dnsv2.TypeA, "192.0.2.1"),
		dc.MustNewRecordConfig("txt", 60, dnsv2.TypeTXT, "hello"),
	}
	prepareDesired(dc)
	var got []string
	for _, rc := range dc.Records {
		got = append(got, rc.StringWithMeta())
	}
	want := []string{
		"sub 300 IN NS ns1.other.example.",
		"www 300 IN A 192.0.2.1",
		"txt 30 IN TXT \"hello\"",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("prepareDesired:\n got %q\nwant %q", got, want)
	}
}

func TestAuditRecords(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	bad := models.Records{
		dc.MustNewRecordConfig("q", 300, dnsv2.TypeTXT, `has "quotes"`),
		dc.MustNewRecordConfig("long", 300, dnsv2.TypeTXT, strings.Repeat("x", 507)),
		dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(1), "issue", "letsencrypt.org"),
		dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(0), "issuemail", "letsencrypt.org"),
		dc.MustNewRecordConfig("_sip._tcp", 300, dnsv2.TypeSRV, uint16(10), uint16(10), uint16(5060), "a.example.com."),
		dc.MustNewRecordConfig("_sip._tcp", 300, dnsv2.TypeSRV, uint16(20), uint16(10), uint16(5060), "b.example.com."),
	}
	errs := (&nameoceanProvider{}).AuditRecords(bad)
	if len(errs) != 5 {
		t.Errorf("AuditRecords returned %d errors, want 5: %v", len(errs), errs)
	}

	good := models.Records{
		dc.MustNewRecordConfig("long", 300, dnsv2.TypeTXT, strings.Repeat("x", 506)),
		dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(128), "issue", "letsencrypt.org"),
		dc.MustNewRecordConfig("_sip._tcp", 300, dnsv2.TypeSRV, uint16(10), uint16(10), uint16(5060), "a.example.com."),
		dc.MustNewRecordConfig("_sip._udp", 300, dnsv2.TypeSRV, uint16(10), uint16(10), uint16(5060), "a.example.com."),
	}
	if errs := (&nameoceanProvider{}).AuditRecords(good); len(errs) != 0 {
		t.Errorf("AuditRecords(good) = %v, want no errors", errs)
	}
}

const exportFixture = `{
  "$schema": "https://nameocean.net/schemas/dns-zone.json",
  "zone": "example.com",
  "records": [
    {"name": "@", "type": "A", "value": "192.0.2.1"},
    {"name": "@", "type": "REDIRECT", "value": "https://example.net/", "save_path": false},
    {"name": "192.0.2.9", "type": "PTR", "value": "host.example.com"},
    {"name": "www", "type": "CNAME", "value": "@", "ttl": "5m"},
    {"name": "old", "type": "TXT", "value": "remove me"}
  ]
}`

// TestPushAppliesWholeZone runs GetZoneRecords and GetZoneRecordsCorrections
// against a fake API and checks the apply request: the serial from the
// export, the desired records, and the REDIRECT/PTR records passed through.
func TestPushAppliesWholeZone(t *testing.T) {
	var applyBody []byte
	var applySerial string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dns/example.com/export":
			if r.URL.Query().Get("output") != "json" {
				t.Errorf("export output = %q, want json", r.URL.Query().Get("output"))
			}
			w.Header().Set("X-Zone-Serial", "2026112001")
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, exportFixture)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/dns/example.com/apply":
			applySerial = r.URL.Query().Get("serial")
			applyBody, _ = io.ReadAll(r.Body)
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("apply Content-Type = %q", ct)
			}
			io.WriteString(w, `{"zone":"example.com","serial":2026112002,"applied":true,"create":[],"update":[],"delete":[],
				"summary":{"create":1,"update":1,"delete":1,"unchanged":1}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := &nameoceanProvider{}
	if err := p.Initialize(map[string]string{"token": "test-token", "apiurl": srv.URL + "/api/v1/"}, nil, nil); err != nil {
		t.Fatal(err)
	}

	dc := models.MustNewDomainConfig("example.com")
	existing, err := p.GetZoneRecords(dc)
	if err != nil {
		t.Fatal(err)
	}
	if len(existing) != 3 {
		t.Fatalf("GetZoneRecords returned %d records, want 3 (REDIRECT and PTR are not managed): %v", len(existing), existing)
	}

	dc.Records = models.Records{
		dc.MustNewRecordConfig("@", 300, dnsv2.TypeNS, "ns1.nameocean.net."),
		dc.MustNewRecordConfig("@", 3600, dnsv2.TypeA, "192.0.2.1"),
		dc.MustNewRecordConfig("www", 1800, dnsv2.TypeCNAME, "example.com."),
		dc.MustNewRecordConfig("@", 600, dnsv2.TypeMX, uint16(10), "mail.example.com."),
	}
	corrections, count, err := p.GetZoneRecordsCorrections(dc, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(corrections) != 1 || count != 3 {
		t.Fatalf("got %d corrections and %d changes, want 1 correction with 3 changes", len(corrections), count)
	}
	if err := corrections[0].F(); err != nil {
		t.Fatal(err)
	}

	if applySerial != "2026112001" {
		t.Errorf("apply serial = %q, want 2026112001", applySerial)
	}
	var got map[string]any
	if err := json.Unmarshal(applyBody, &got); err != nil {
		t.Fatalf("apply body is not JSON: %v\n%s", err, applyBody)
	}
	want := map[string]any{
		"zone": "example.com",
		"records": []any{
			map[string]any{"name": "@", "type": "A", "value": "192.0.2.1", "ttl": "1h"},
			map[string]any{"name": "www", "type": "CNAME", "value": "@", "ttl": "30m"},
			map[string]any{"name": "@", "type": "MX", "value": "mail.example.com", "priority": 10.0, "ttl": "5m"},
			map[string]any{"name": "@", "type": "REDIRECT", "value": "https://example.net/", "save_path": false},
			map[string]any{"name": "192.0.2.9", "type": "PTR", "value": "host.example.com"},
		},
	}
	gotJSON, _ := json.MarshalIndent(got, "", "  ")
	wantJSON, _ := json.MarshalIndent(want, "", "  ")
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("apply body:\n%s\nwant:\n%s", gotJSON, wantJSON)
	}
}

func TestApplyZoneChanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		io.WriteString(w, `{"error":"The zone changed since the plan (serial 1, now 2). Run the plan again.","code":"zone_changed"}`)
	}))
	defer srv.Close()

	p := &nameoceanProvider{}
	if err := p.Initialize(map[string]string{"token": "t", "apiurl": srv.URL}, nil, nil); err != nil {
		t.Fatal(err)
	}
	_, err := p.applyZone("example.com", "1", &zoneDocument{Zone: "example.com"})
	if err == nil || !strings.Contains(err.Error(), "zone_changed") {
		t.Errorf("applyZone error = %v, want zone_changed", err)
	}
}

func TestEnsureZoneExistsAlreadyExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/dns" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		w.WriteHeader(http.StatusConflict)
		io.WriteString(w, `{"error":"A DNS zone for example.com already exists.","code":"already_exists"}`)
	}))
	defer srv.Close()

	p := &nameoceanProvider{}
	if err := p.Initialize(map[string]string{"token": "t", "apiurl": srv.URL}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := p.EnsureZoneExists(models.MustNewDomainConfig("example.com")); err != nil {
		t.Errorf("EnsureZoneExists = %v, want nil", err)
	}
}

func TestListZones(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"zones":[{"name":"example.com","serial":12,"dnssec":false},{"name":"Example.ORG","serial":3,"dnssec":true}]}`)
	}))
	defer srv.Close()

	p := &nameoceanProvider{}
	if err := p.Initialize(map[string]string{"token": "t", "apiurl": srv.URL}, nil, nil); err != nil {
		t.Fatal(err)
	}
	zones, err := p.ListZones()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(zones, ",") != "example.com,example.org" {
		t.Errorf("ListZones = %v", zones)
	}
}

func TestDefinition(t *testing.T) {
	def, ok := providers.GetDefinition("NAMEOCEAN")
	if !ok {
		t.Fatal("NAMEOCEAN is not registered")
	}
	if !def.CanGetZones || !def.DocCreateDomains {
		t.Errorf("CanGetZones = %v, DocCreateDomains = %v; want both true", def.CanGetZones, def.DocCreateDomains)
	}
}

func TestRateLimitedRequestIsRetried(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"zone":"example.com"`) {
			t.Errorf("attempt %d sent body %q", calls, body)
		}
		if calls == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			io.WriteString(w, `{"error":"Request was throttled.","code":"rate_limited"}`)
			return
		}
		io.WriteString(w, `{"zone":"example.com","serial":2,"applied":true}`)
	}))
	defer srv.Close()

	p := &nameoceanProvider{}
	if err := p.Initialize(map[string]string{"token": "t", "apiurl": srv.URL}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.applyZone("example.com", "1", &zoneDocument{Zone: "example.com"}); err != nil {
		t.Fatalf("applyZone: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestRetryAfter(t *testing.T) {
	for in, want := range map[string]time.Duration{"": 5 * time.Second, "3": 3 * time.Second, "999": 65 * time.Second, "x": 5 * time.Second} {
		if got := retryAfter(in); got != want {
			t.Errorf("retryAfter(%q) = %s, want %s", in, got, want)
		}
	}
}
