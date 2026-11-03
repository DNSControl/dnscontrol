package ionos

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/zonecache"
)

const testZoneID = "11af3414-ebba-11e9-8df5-66fbe8a334b4"

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   string
}

// fakeAPI serves the IONOS DNS API routes the provider uses and records every
// write it receives.
type fakeAPI struct {
	t        *testing.T
	records  []record
	requests []recordedRequest
	apiKeys  []string

	// deleteErrors maps a record ID to the error body that a DELETE of that
	// record answers with, as a 404.
	deleteErrors map[string]string
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		f.t.Errorf("reading request body: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f.requests = append(f.requests, recordedRequest{r.Method, r.URL.Path, r.URL.RawQuery, string(body)})
	f.apiKeys = append(f.apiKeys, r.Header.Get("X-API-Key"))
	w.Header().Set("Content-Type", "application/json")

	zonePath := "/zones/" + testZoneID
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/zones":
		_, _ = io.WriteString(w, `[{"id":"`+testZoneID+`","name":"example.com","type":"NATIVE"}]`)
	case r.Method == http.MethodGet && r.URL.Path == zonePath:
		if err := json.NewEncoder(w).Encode(zoneDetail{ID: testZoneID, Name: "example.com", Type: "NATIVE", Records: f.records}); err != nil {
			f.t.Errorf("encoding zone: %v", err)
		}
	case r.Method == http.MethodPost && r.URL.Path == zonePath+"/records":
		_, _ = io.WriteString(w, `[]`)
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, zonePath+"/records/"):
		_, _ = io.WriteString(w, `{}`)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, zonePath+"/records/"):
		if body, ok := f.deleteErrors[strings.TrimPrefix(r.URL.Path, zonePath+"/records/")]; ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, body)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `[{"code":"RECORD_NOT_FOUND","message":"Record does not exist."}]`)
	}
}

// writes returns the recorded requests that are not reads.
func (f *fakeAPI) writes() []recordedRequest {
	var out []recordedRequest
	for _, r := range f.requests {
		if r.Method != http.MethodGet {
			out = append(out, r)
		}
	}
	return out
}

func newTestProvider(t *testing.T, records ...record) (*ionosProvider, *fakeAPI) {
	t.Helper()
	api := &fakeAPI{t: t, records: records}
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)
	return newProviderFor(srv), api
}

func newProviderFor(srv *httptest.Server) *ionosProvider {
	p := &ionosProvider{client: srv.Client(), apiKey: "prefix.secret", baseURL: srv.URL, sleep: func(time.Duration) {}}
	p.zoneCache = zonecache.New(p.fetchAllZones)
	return p
}

func newTestDomain(t *testing.T) *models.DomainConfig {
	t.Helper()
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	return dc
}

func TestInitializeRequiresAPIKey(t *testing.T) {
	p := &ionosProvider{}
	if err := p.Initialize(map[string]string{}, nil, nil); err == nil {
		t.Fatal("Initialize accepted creds without api_key")
	}
	if err := p.Initialize(map[string]string{"api_key": "prefix.secret"}, nil, nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if p.baseURL != "https://api.hosting.ionos.com/dns/v1" {
		t.Errorf("baseURL = %q", p.baseURL)
	}
}

func TestToRC(t *testing.T) {
	dc := newTestDomain(t)
	tests := []struct {
		name      string
		in        record
		wantLabel string
		wantRDATA string
	}{
		{"A at apex", record{Name: "example.com", Type: "A", Content: "192.0.2.1", TTL: 3600}, "@", "192.0.2.1"},
		{"AAAA", record{Name: "www.example.com", Type: "AAAA", Content: "2001:db8::1", TTL: 3600}, "www", "2001:db8::1"},
		{"CNAME without dot", record{Name: "www.example.com", Type: "CNAME", Content: "target.example.net", TTL: 3600}, "www", "target.example.net."},
		{"NS without dot", record{Name: "sub.example.com", Type: "NS", Content: "ns1.example.net", TTL: 3600}, "sub", "ns1.example.net."},
		{"MX priority lives in prio", record{Name: "example.com", Type: "MX", Content: "mx.example.net", TTL: 3600, Prio: 10}, "@", "10 mx.example.net."},
		{"SRV priority lives in prio", record{Name: "_sip._tcp.example.com", Type: "SRV", Content: "5 5060 sip.example.net", TTL: 3600, Prio: 10}, "_sip._tcp", "10 5 5060 sip.example.net."},
		{"SRV priority lives in content", record{Name: "_sip._tcp.example.com", Type: "SRV", Content: "10 5 5060 sip.example.net", TTL: 3600}, "_sip._tcp", "10 5 5060 sip.example.net."},
		{"TXT quoted by the API", record{Name: "example.com", Type: "TXT", Content: `"v=spf1 -all"`, TTL: 3600}, "@", `"v=spf1 -all"`},
		{"TXT unquoted", record{Name: "example.com", Type: "TXT", Content: "v=spf1 -all", TTL: 3600}, "@", `"v=spf1 -all"`},
		{"CAA", record{Name: "example.com", Type: "CAA", Content: `0 issue "letsencrypt.org"`, TTL: 3600}, "@", `0 issue "letsencrypt.org"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := toRC(dc, tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetLabel(); got != tc.wantLabel {
				t.Errorf("label = %q, want %q", got, tc.wantLabel)
			}
			if got := rc.GetRDATA().String(); got != tc.wantRDATA {
				t.Errorf("rdata = %q, want %q", got, tc.wantRDATA)
			}
			if rc.TTL != 3600 {
				t.Errorf("TTL = %d, want 3600", rc.TTL)
			}
		})
	}
}

func TestToNative(t *testing.T) {
	dc := newTestDomain(t)
	tests := []struct {
		name string
		rc   *models.RecordConfig
		want record
	}{
		{"A at apex", dc.MustNewRecordConfigParse("@", 300, "A", "192.0.2.1"), record{Name: "example.com", Type: "A", Content: "192.0.2.1", TTL: 300}},
		{"CNAME loses its dot", dc.MustNewRecordConfigParse("www", 300, "CNAME", "target.example.net."), record{Name: "www.example.com", Type: "CNAME", Content: "target.example.net", TTL: 300}},
		{"NS loses its dot", dc.MustNewRecordConfigParse("sub", 300, "NS", "ns1.example.net."), record{Name: "sub.example.com", Type: "NS", Content: "ns1.example.net", TTL: 300}},
		{"MX splits priority", dc.MustNewRecordConfigParse("@", 300, "MX", "10 mx.example.net."), record{Name: "example.com", Type: "MX", Content: "mx.example.net", TTL: 300, Prio: 10}},
		{"SRV splits priority", dc.MustNewRecordConfigParse("_sip._tcp", 300, "SRV", "10 5 5060 sip.example.net."), record{Name: "_sip._tcp.example.com", Type: "SRV", Content: "5 5060 sip.example.net", TTL: 300, Prio: 10}},
		{"TXT is sent unquoted", dc.MustNewRecordConfigParse("@", 300, "TXT", `"v=spf1 -all"`), record{Name: "example.com", Type: "TXT", Content: "v=spf1 -all", TTL: 300}},
		{"CAA", dc.MustNewRecordConfigParse("@", 300, "CAA", `0 issue "letsencrypt.org"`), record{Name: "example.com", Type: "CAA", Content: `0 issue "letsencrypt.org"`, TTL: 300}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := toNative(tc.rc)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("toNative = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestGetZoneRecordsSkipsSOAAndSendsAPIKey(t *testing.T) {
	p, api := newTestProvider(t,
		record{ID: "soa", Name: "example.com", Type: "SOA", Content: "ns1.example.net. hostmaster.example.com. 1 2 3 4 5", TTL: 3600},
		record{ID: "a", Name: "www.example.com", Type: "A", Content: "192.0.2.1", TTL: 3600},
	)
	got, err := p.GetZoneRecords(newTestDomain(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].GetLabel() != "www" {
		t.Fatalf("records = %v, want only www", got)
	}
	for _, key := range api.apiKeys {
		if key != "prefix.secret" {
			t.Errorf("X-API-Key = %q", key)
		}
	}
}

func TestGetZoneRecordsUnknownZone(t *testing.T) {
	p, _ := newTestProvider(t)
	dc, err := models.NewDomainConfig("absent.example")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GetZoneRecords(dc)
	if err == nil || !strings.Contains(err.Error(), "absent.example") {
		t.Fatalf("err = %v, want an error naming the zone", err)
	}
}

// applyCorrections runs the whole path: read the zone, diff it against dc, apply.
func applyCorrections(t *testing.T, p *ionosProvider, dc *models.DomainConfig) int {
	t.Helper()
	existing, err := p.GetZoneRecords(dc)
	if err != nil {
		t.Fatal(err)
	}
	corrections, _, err := p.GetZoneRecordsCorrections(dc, existing)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range corrections {
		if c.F == nil {
			continue
		}
		if err := c.F(); err != nil {
			t.Fatalf("%s: %v", c.Msg, err)
		}
	}
	return len(corrections)
}

func TestCorrectionsCreateChangeDelete(t *testing.T) {
	p, api := newTestProvider(t,
		record{ID: "keep", Name: "keep.example.com", Type: "A", Content: "192.0.2.1", TTL: 300},
		record{ID: "chg", Name: "chg.example.com", Type: "A", Content: "192.0.2.2", TTL: 300},
		record{ID: "del", Name: "del.example.com", Type: "A", Content: "192.0.2.3", TTL: 300},
	)
	dc := newTestDomain(t)
	addRecord(dc, "keep", 300, "A", "192.0.2.1")
	addRecord(dc, "chg", 600, "A", "192.0.2.20")
	addRecord(dc, "new", 300, "MX", "10 mx.example.net.")

	applyCorrections(t, p, dc)

	got := map[string]recordedRequest{}
	for _, w := range api.writes() {
		got[w.Method] = w
	}
	if len(api.writes()) != 3 {
		t.Fatalf("writes = %+v, want exactly one POST, one PUT, one DELETE", api.writes())
	}
	zonePath := "/zones/" + testZoneID + "/records"

	post := got[http.MethodPost]
	if post.Path != zonePath {
		t.Errorf("POST path = %q", post.Path)
	}
	assertJSON(t, post.Body, `[{"name":"new.example.com","type":"MX","content":"mx.example.net","ttl":300,"prio":10,"disabled":false}]`)

	put := got[http.MethodPut]
	if put.Path != zonePath+"/chg" {
		t.Errorf("PUT path = %q", put.Path)
	}
	assertJSON(t, put.Body, `{"content":"192.0.2.20","ttl":600,"prio":0,"disabled":false}`)

	if del := got[http.MethodDelete]; del.Path != zonePath+"/del" {
		t.Errorf("DELETE path = %q", del.Path)
	}
}

func TestCorrectionsNoChangeIsSilent(t *testing.T) {
	p, api := newTestProvider(t,
		record{ID: "txt", Name: "example.com", Type: "TXT", Content: `"v=spf1 -all"`, TTL: 300},
		record{ID: "mx", Name: "example.com", Type: "MX", Content: "mx.example.net", TTL: 300, Prio: 10},
	)
	dc := newTestDomain(t)
	addRecord(dc, "@", 300, "TXT", `"v=spf1 -all"`)
	addRecord(dc, "@", 300, "MX", "10 mx.example.net.")

	if n := applyCorrections(t, p, dc); n != 0 {
		t.Errorf("corrections = %d, want 0", n)
	}
	if w := api.writes(); len(w) != 0 {
		t.Errorf("writes = %+v, want none", w)
	}
}

// IONOS deactivates records that conflict with its services. A disabled record
// is not served, so it must not count as already matching the desired zone.
func TestCorrectionsReenableDisabledRecord(t *testing.T) {
	p, api := newTestProvider(t,
		record{ID: "off", Name: "www.example.com", Type: "A", Content: "192.0.2.1", TTL: 300, Disabled: true},
	)
	dc := newTestDomain(t)
	addRecord(dc, "www", 300, "A", "192.0.2.1")

	applyCorrections(t, p, dc)

	w := api.writes()
	if len(w) != 1 || w[0].Method != http.MethodPut || !strings.HasSuffix(w[0].Path, "/records/off") {
		t.Fatalf("writes = %+v, want one PUT on the disabled record", w)
	}
	assertJSON(t, w[0].Body, `{"content":"192.0.2.1","ttl":300,"prio":0,"disabled":false}`)
}

func TestGetNameserversReadsEnabledApexNS(t *testing.T) {
	p, api := newTestProvider(t,
		record{ID: "1", Name: "example.com", Type: "NS", Content: "ns1045.ui-dns.com", TTL: 86400},
		record{ID: "2", Name: "example.com", Type: "NS", Content: "ns1102.ui-dns.org.", TTL: 86400},
		record{ID: "3", Name: "example.com", Type: "NS", Content: "off.ui-dns.net", TTL: 86400, Disabled: true},
		record{ID: "4", Name: "sub.example.com", Type: "NS", Content: "delegated.example.net", TTL: 86400},
		record{ID: "5", Name: "example.com", Type: "A", Content: "192.0.2.1", TTL: 86400},
	)
	got, err := p.GetNameservers("example.com")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, ns := range got {
		names = append(names, ns.Name)
	}
	want := []string{"ns1045.ui-dns.com", "ns1102.ui-dns.org"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("nameservers = %v, want %v", names, want)
	}
	if len(api.requests) == 0 {
		t.Fatal("no request sent")
	}
}

func TestListZones(t *testing.T) {
	p, _ := newTestProvider(t)
	got, err := p.ListZones()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"example.com"}) {
		t.Errorf("zones = %v", got)
	}
}

func TestAPIErrorCarriesCodeAndMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `[{"code":"UNAUTHORIZED","message":"The customer is not authorized to do this operation."}]`)
	}))
	defer srv.Close()
	p := newProviderFor(srv)

	_, err := p.ListZones()
	if err == nil {
		t.Fatal("no error on HTTP 401")
	}
	for _, want := range []string{"401", "UNAUTHORIZED", "not authorized"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "prefix.secret") {
		t.Errorf("error %q leaks the API key", err)
	}
}

func TestAuditRecords(t *testing.T) {
	dc := newTestDomain(t)
	p := &ionosProvider{}
	if errs := p.AuditRecords(models.Records{dc.MustNewRecordConfigParse("www", 60, "A", "192.0.2.1")}); len(errs) != 0 {
		t.Errorf("TTL 60 rejected: %v", errs)
	}
	if errs := p.AuditRecords(models.Records{dc.MustNewRecordConfigParse("www", 59, "A", "192.0.2.1")}); len(errs) != 1 {
		t.Errorf("TTL 59 not rejected: %v", errs)
	}
	rejected := map[string]*models.RecordConfig{
		"null MX":        dc.MustNewRecordConfigParse("@", 300, "MX", "0 ."),
		"null SRV":       dc.MustNewRecordConfigParse("_sip._tcp", 300, "SRV", "0 0 0 ."),
		"root CNAME":     dc.MustNewRecordConfigParse("a", 300, "CNAME", "."),
		"root NS":        dc.MustNewRecordConfigParse("sub", 300, "NS", "."),
		"TXT with quote": dc.MustNewRecordConfig("a", 300, "TXT", `say "hi"`),
		"TXT backslash":  dc.MustNewRecordConfig("a", 300, "TXT", `back\slash`),
		"empty TXT":      dc.MustNewRecordConfig("a", 300, "TXT", ""),
	}
	for name, rc := range rejected {
		if errs := p.AuditRecords(models.Records{rc}); len(errs) != 1 {
			t.Errorf("%s not rejected: %v", name, errs)
		}
	}
}

func assertJSON(t *testing.T, got, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Fatalf("body %q: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("expected %q: %v", want, err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func addRecord(dc *models.DomainConfig, label string, ttl uint32, rtype, data string) {
	dc.Records = append(dc.Records, dc.MustNewRecordConfigParse(label, ttl, rtype, data))
}

func TestCorrectionsChangeSendsPriority(t *testing.T) {
	p, api := newTestProvider(t,
		record{ID: "mx", Name: "example.com", Type: "MX", Content: "mx.example.net", TTL: 300, Prio: 10},
	)
	dc := newTestDomain(t)
	addRecord(dc, "@", 300, "MX", "20 mx.example.net.")

	applyCorrections(t, p, dc)

	w := api.writes()
	if len(w) != 1 || w[0].Method != http.MethodPut {
		t.Fatalf("writes = %+v, want one PUT", w)
	}
	assertJSON(t, w[0].Body, `{"content":"mx.example.net","ttl":300,"prio":20,"disabled":false}`)
}

// The API documents a list of errors, but answers an invalid key with a single object.
func TestAPIErrorKeepsUndocumentedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"message":"Missing or invalid API key."}`)
	}))
	defer srv.Close()
	p := newProviderFor(srv)

	_, err := p.ListZones()
	if err == nil || !strings.Contains(err.Error(), "Missing or invalid API key.") {
		t.Fatalf("err = %v, want the body of the response", err)
	}
}

// IONOS manages the apex NS records. Neither a declaration that differs from
// them nor the absence of a declaration may rewrite or delete them.
func TestCorrectionsLeaveApexNSAlone(t *testing.T) {
	apexNS := []record{
		{ID: "ns1", Name: "example.com", Type: "NS", Content: "ns1045.ui-dns.com", TTL: 172800},
		{ID: "ns2", Name: "example.com", Type: "NS", Content: "ns1045.ui-dns.org", TTL: 172800},
	}

	t.Run("not declared", func(t *testing.T) {
		p, api := newTestProvider(t, apexNS...)
		applyCorrections(t, p, newTestDomain(t))
		if w := api.writes(); len(w) != 0 {
			t.Errorf("writes = %+v, want none", w)
		}
	})

	t.Run("declared with another TTL", func(t *testing.T) {
		p, api := newTestProvider(t, apexNS...)
		dc := newTestDomain(t)
		addRecord(dc, "@", 300, "NS", "ns1045.ui-dns.com.")
		addRecord(dc, "@", 300, "NS", "ns1045.ui-dns.org.")
		applyCorrections(t, p, dc)
		if w := api.writes(); len(w) != 0 {
			t.Errorf("writes = %+v, want none", w)
		}
	})

	t.Run("delegations stay managed", func(t *testing.T) {
		p, api := newTestProvider(t, record{ID: "sub", Name: "sub.example.com", Type: "NS", Content: "ns1.example.net", TTL: 3600})
		applyCorrections(t, p, newTestDomain(t))
		w := api.writes()
		if len(w) != 1 || w[0].Method != http.MethodDelete || !strings.HasSuffix(w[0].Path, "/records/sub") {
			t.Errorf("writes = %+v, want one DELETE of the delegation", w)
		}
	})
}

// Record types the provider cannot declare must survive a push, like the SOA.
func TestCorrectionsLeaveUnsupportedTypesAlone(t *testing.T) {
	p, api := newTestProvider(t,
		record{ID: "tlsa", Name: "_443._tcp.example.com", Type: "TLSA", Content: "3 1 1 abcd", TTL: 3600},
		record{ID: "svcb", Name: "svc.example.com", Type: "SVCB", Content: "1 svc.example.net alpn=h3", TTL: 3600},
	)
	applyCorrections(t, p, newTestDomain(t))
	if w := api.writes(); len(w) != 0 {
		t.Errorf("writes = %+v, want none", w)
	}
}

func TestZoneListIsFetchedOnce(t *testing.T) {
	p, api := newTestProvider(t, record{ID: "a", Name: "www.example.com", Type: "A", Content: "192.0.2.1", TTL: 3600})
	dc := newTestDomain(t)
	addRecord(dc, "www", 3600, "A", "192.0.2.2")
	if _, err := p.GetNameservers("example.com"); err != nil {
		t.Fatal(err)
	}
	applyCorrections(t, p, dc)

	fetches := 0
	for _, r := range api.requests {
		if r.Method == http.MethodGet && r.Path == "/zones" {
			fetches++
		}
	}
	if fetches != 1 {
		t.Errorf("GET /zones sent %d times, want 1", fetches)
	}
}

// A record the provider cannot parse must not stop the whole zone: it is left
// out of the diff, so it is neither rewritten nor deleted, and the operator is told.
func TestUnparsableRecordIsSkippedWithAWarning(t *testing.T) {
	tests := []struct {
		name   string
		bad    record
		naming string
	}{
		{
			name:   "unterminated TXT",
			bad:    record{ID: "bad", Name: "bad.example.com", Type: "TXT", Content: `"unterminated`, TTL: 3600},
			naming: "bad.example.com",
		},
		{
			name:   "SRV with two fields",
			bad:    record{ID: "bad", Name: "_sip._tcp.example.com", Type: "SRV", Content: "1 2", TTL: 3600},
			naming: "_sip._tcp.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var warnings bytes.Buffer
			previous := printer.DefaultPrinter
			printer.DefaultPrinter = &printer.ConsolePrinter{Reader: bufio.NewReader(strings.NewReader("")), Writer: &warnings}
			t.Cleanup(func() { printer.DefaultPrinter = previous })

			p, api := newTestProvider(t,
				tt.bad,
				record{ID: "a", Name: "www.example.com", Type: "A", Content: "192.0.2.1", TTL: 300},
			)
			dc := newTestDomain(t)
			addRecord(dc, "www", 300, "A", "192.0.2.1")

			existing, err := p.GetZoneRecords(dc)
			if err != nil {
				t.Fatalf("GetZoneRecords: %v", err)
			}
			if len(existing) != 1 || existing[0].GetLabel() != "www" {
				t.Fatalf("records = %v, want only www", existing)
			}
			if !strings.Contains(warnings.String(), tt.naming) {
				t.Errorf("warnings = %q, want them to name %q", warnings.String(), tt.naming)
			}

			applyCorrections(t, p, dc)
			if w := api.writes(); len(w) != 0 {
				t.Errorf("writes = %+v, want none", w)
			}
		})
	}
}

// rateLimitedServer answers HTTP 429 to the first failures requests, then 200,
// and records every body it receives.
func rateLimitedServer(t *testing.T, failures int, retryAfter string) (*ionosProvider, *[]string, *[]time.Duration) {
	t.Helper()
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
		}
		bodies = append(bodies, string(body))
		if len(bodies) <= failures {
			if retryAfter != "" {
				w.Header().Set("Retry-After", retryAfter)
			}
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `[{"code":"TOO_MANY_REQUESTS","message":"Slow down."}]`)
			return
		}
		_, _ = io.WriteString(w, `[]`)
	}))
	t.Cleanup(srv.Close)

	var waits []time.Duration
	p := newProviderFor(srv)
	p.sleep = func(d time.Duration) { waits = append(waits, d) }
	return p, &bodies, &waits
}

func TestRateLimitedWriteIsResentWithItsBody(t *testing.T) {
	p, bodies, waits := rateLimitedServer(t, 2, "")

	r := record{Name: "www.example.com", Type: "A", Content: "192.0.2.1", TTL: 300}
	if err := p.createRecord(testZoneID, r); err != nil {
		t.Fatalf("createRecord: %v", err)
	}

	want := `[{"name":"www.example.com","type":"A","content":"192.0.2.1","ttl":300,"prio":0,"disabled":false}]`
	if len(*bodies) != 3 {
		t.Fatalf("requests = %d, want 3", len(*bodies))
	}
	for _, body := range *bodies {
		assertJSON(t, body, want)
	}
	if !reflect.DeepEqual(*waits, []time.Duration{5 * time.Second, 5 * time.Second}) {
		t.Errorf("waits = %v, want [5s 5s]", *waits)
	}
}

func TestRateLimitHonoursALongerRetryAfter(t *testing.T) {
	p, _, waits := rateLimitedServer(t, 1, "20")

	if err := p.deleteRecord(testZoneID, "id"); err != nil {
		t.Fatalf("deleteRecord: %v", err)
	}
	if !reflect.DeepEqual(*waits, []time.Duration{20 * time.Second}) {
		t.Errorf("waits = %v, want [20s]", *waits)
	}
}

func TestRateLimitGivesUpAfterTheMaximumWait(t *testing.T) {
	p, bodies, waits := rateLimitedServer(t, 1000, "")

	err := p.deleteRecord(testZoneID, "id")
	if err == nil {
		t.Fatal("no error after the rate limit outlasted the maximum wait")
	}
	for _, want := range []string{"429", "TOO_MANY_REQUESTS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
	var total time.Duration
	for _, wait := range *waits {
		total += wait
	}
	if total > 3*time.Minute {
		t.Errorf("waited %v, want at most 3m", total)
	}
	if len(*bodies) != len(*waits)+1 {
		t.Errorf("requests = %d after %d waits, want one more request than waits", len(*bodies), len(*waits))
	}
}

func TestOtherErrorsAreNotRetried(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	p := newProviderFor(srv)

	if err := p.deleteRecord(testZoneID, "id"); err == nil {
		t.Fatal("no error on HTTP 500")
	}
	if requests != 1 {
		t.Errorf("requests = %d, want 1", requests)
	}
}

// A disabled record that dnsconfig.js does not declare is deleted like any other
// undeclared record: documentation/provider/ionos.md warns about it.
func TestCorrectionsDeleteUndeclaredDisabledRecord(t *testing.T) {
	p, api := newTestProvider(t,
		record{ID: "off", Name: "old.example.com", Type: "A", Content: "192.0.2.1", TTL: 300, Disabled: true},
	)

	applyCorrections(t, p, newTestDomain(t))

	w := api.writes()
	if len(w) != 1 || w[0].Method != http.MethodDelete || !strings.HasSuffix(w[0].Path, "/records/off") {
		t.Fatalf("writes = %+v, want one DELETE on the disabled record", w)
	}
}

// A skipped record is not among the existing records, so declaring the same value
// must not create a duplicate of it. Another value at that name is a normal create.
func TestDeclaringASkippedRecord(t *testing.T) {
	bad := record{ID: "bad", Name: "bad.example.com", Type: "TXT", Content: `"unterminated`, TTL: 3600}

	t.Run("same value is left alone", func(t *testing.T) {
		var warnings bytes.Buffer
		previous := printer.DefaultPrinter
		printer.DefaultPrinter = &printer.ConsolePrinter{Reader: bufio.NewReader(strings.NewReader("")), Writer: &warnings}
		t.Cleanup(func() { printer.DefaultPrinter = previous })

		p, api := newTestProvider(t, bad)
		dc := newTestDomain(t)
		addRecord(dc, "bad", 3600, "TXT", `"unterminated"`)

		applyCorrections(t, p, dc)

		if w := api.writes(); len(w) != 0 {
			t.Errorf("writes = %+v, want none", w)
		}
		if got := strings.Count(warnings.String(), "bad.example.com"); got != 2 {
			t.Errorf("warnings = %q, want one for the skip and one for the declaration", warnings.String())
		}
	})

	t.Run("another value is created", func(t *testing.T) {
		p, api := newTestProvider(t, bad)
		dc := newTestDomain(t)
		addRecord(dc, "bad", 3600, "TXT", `"valid"`)

		applyCorrections(t, p, dc)

		w := api.writes()
		if len(w) != 1 || w[0].Method != http.MethodPost {
			t.Fatalf("writes = %+v, want one POST", w)
		}
		assertJSON(t, w[0].Body, `[{"name":"bad.example.com","type":"TXT","content":"valid","ttl":3600,"prio":0,"disabled":false}]`)
	})
}

// A CNAME cannot share its name with another record. A skipped record is invisible
// to the diff, so such a conflict must be reported before anything is written.
func TestDeclarationConflictingWithASkippedRecord(t *testing.T) {
	tests := []struct {
		name         string
		bad          record
		rtype, value string
	}{
		{
			name:  "TXT declared on a skipped CNAME",
			bad:   record{ID: "bad", Name: "bad.example.com", Type: "CNAME", Content: "not a name", TTL: 3600},
			rtype: "TXT", value: `"valid"`,
		},
		{
			name:  "another CNAME declared on a skipped CNAME",
			bad:   record{ID: "bad", Name: "bad.example.com", Type: "CNAME", Content: "not a name", TTL: 3600},
			rtype: "CNAME", value: "target.example.net.",
		},
		{
			name:  "CNAME declared on a skipped TXT",
			bad:   record{ID: "bad", Name: "bad.example.com", Type: "TXT", Content: `"unterminated`, TTL: 3600},
			rtype: "CNAME", value: "target.example.net.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, api := newTestProvider(t, tt.bad)
			dc := newTestDomain(t)
			addRecord(dc, "bad", 3600, tt.rtype, tt.value)

			existing, err := p.GetZoneRecords(dc)
			if err != nil {
				t.Fatalf("GetZoneRecords: %v", err)
			}
			if len(existing) != 0 {
				t.Fatalf("records = %v, want the bad record skipped", existing)
			}
			_, _, err = p.GetZoneRecordsCorrections(dc, existing)
			if err == nil {
				t.Fatal("no error on a declaration that conflicts with a skipped record")
			}
			if !strings.Contains(err.Error(), "bad.example.com") {
				t.Errorf("error %q does not name the record", err)
			}
			if w := api.writes(); len(w) != 0 {
				t.Errorf("writes = %+v, want none", w)
			}
		})
	}
}

// IONOS deletes some records along with others: removing the apex A record of a
// zone bound to an IONOS website also removes the apex AAAA, the www records and
// their _dep_ws_mutex TXT. The DELETE of a record already removed that way answers
// 404 RECORD_NOT_FOUND, and the push must carry on.
func TestCorrectionsDeleteOfRecordAlreadyGoneSucceeds(t *testing.T) {
	p, api := newTestProvider(t,
		record{ID: "apex-a", Name: "example.com", Type: "A", Content: "192.0.2.1", TTL: 3600},
		record{ID: "apex-aaaa", Name: "example.com", Type: "AAAA", Content: "2001:db8::1", TTL: 3600},
	)
	api.deleteErrors = map[string]string{
		"apex-aaaa": `[{"code":"RECORD_NOT_FOUND","message":"Record does not exist."}]`,
	}
	dc := newTestDomain(t)

	applyCorrections(t, p, dc)

	if got := len(api.writes()); got != 2 {
		t.Fatalf("writes = %+v, want the two DELETE", api.writes())
	}
}

func TestCorrectionsDeleteKeepsOtherNotFoundErrors(t *testing.T) {
	p, api := newTestProvider(t,
		record{ID: "apex-a", Name: "example.com", Type: "A", Content: "192.0.2.1", TTL: 3600},
	)
	api.deleteErrors = map[string]string{
		"apex-a": `[{"code":"ZONE_NOT_FOUND","message":"Zone does not exist."}]`,
	}
	dc := newTestDomain(t)
	existing, err := p.GetZoneRecords(dc)
	if err != nil {
		t.Fatal(err)
	}
	corrections, _, err := p.GetZoneRecordsCorrections(dc, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(corrections) != 1 || corrections[0].F == nil {
		t.Fatalf("corrections = %+v, want one DELETE", corrections)
	}
	err = corrections[0].F()
	if err == nil {
		t.Fatal("a 404 ZONE_NOT_FOUND on DELETE was swallowed")
	}
	if !strings.Contains(err.Error(), "ZONE_NOT_FOUND") {
		t.Errorf("error %q does not carry the API code", err)
	}
}

// RFC 9110 allows Retry-After as an HTTP-date as well as a number of seconds.
func TestRetryAfterAcceptsHTTPDate(t *testing.T) {
	header := http.Header{}
	header.Set("Retry-After", time.Now().Add(60*time.Second).UTC().Format(http.TimeFormat))
	got := retryAfter(header)
	if got < 55*time.Second || got > 60*time.Second {
		t.Errorf("retryAfter(HTTP-date 60s ahead) = %v, want about 60s", got)
	}

	header.Set("Retry-After", time.Now().Add(-time.Minute).UTC().Format(http.TimeFormat))
	if got := retryAfter(header); got != 0 {
		t.Errorf("retryAfter(HTTP-date in the past) = %v, want 0", got)
	}
}
