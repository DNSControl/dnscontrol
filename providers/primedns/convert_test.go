package primedns

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func newDC(t *testing.T) *models.DomainConfig {
	t.Helper()
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	return dc
}

func TestSkipRecord(t *testing.T) {
	cases := []struct {
		name string
		r    record
		want bool
	}{
		{"empty non-terminal", record{Name: "_domainkey.example.com", Type: "", TTL: 0}, true},
		{"soa", record{Name: "example.com", Type: "SOA"}, true},
		{"cds", record{Name: "example.com", Type: "CDS"}, true},
		{"uri", record{Name: "_u.example.com", Type: "URI"}, true},
		{"own apex NS", record{Name: "example.com", Type: "NS", Content: "ns1.no-ip.pl"}, true},
		{"own apex NS primedns", record{Name: "example.com", Type: "NS", Content: "ns2.primedns.net."}, true},
		{"foreign apex NS is kept", record{Name: "example.com", Type: "NS", Content: "ns1.other.example"}, false},
		{"NS below the apex is kept", record{Name: "sub.example.com", Type: "NS", Content: "ns1.no-ip.pl"}, false},
		{"A", record{Name: "www.example.com", Type: "A", Content: "192.0.2.1"}, false},
	}
	for _, c := range cases {
		if got := skipRecord("example.com", c.r); got != c.want {
			t.Errorf("%s: skipRecord = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestTXTToPresentation(t *testing.T) {
	cases := map[string]string{
		`v=spf1 -all`:       `"v=spf1 -all"`,
		`"hello world"`:     `"hello world"`,
		`p=AAAA" "BBBB`:     `"p=AAAA" "BBBB"`,
		`"a" "b"`:           `"a" "b"`,
		`say \"hi\"`:        `"say \"hi\""`,
		``:                  `""`,
		`k=rsa; p=MIIB==`:   `"k=rsa; p=MIIB=="`,
		`"part1" "part2"`:   `"part1" "part2"`,
		`already "quoted" `: `"already "quoted" "`, // unchanged semantics: PowerDNS wraps text that does not start with a quote
	}
	for in, want := range cases {
		if got := txtToPresentation(in); got != want {
			t.Errorf("txtToPresentation(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToRecordConfigLabelsAndTypes(t *testing.T) {
	dc := newDC(t)
	cases := []struct {
		r         record
		wantLabel string
		wantRDATA string
	}{
		{record{ID: 1, Name: "example.com", Type: "A", Content: "192.0.2.1", TTL: 300}, "@", "192.0.2.1"},
		{record{ID: 2, Name: "www.example.com", Type: "CNAME", Content: "example.com", TTL: 300}, "www", "example.com."},
		{record{ID: 3, Name: "*.example.com", Type: "A", Content: "192.0.2.2", TTL: 300}, "*", "192.0.2.2"},
		{record{ID: 4, Name: "example.com", Type: "MX", Content: "mail.example.com", Prio: new(10), TTL: 300}, "@", "10 mail.example.com."},
		{record{ID: 5, Name: "_sip._tcp.example.com", Type: "SRV", Content: "5 5060 sip.example.com", Prio: new(10), TTL: 300}, "_sip._tcp", "10 5 5060 sip.example.com."},
		{record{ID: 6, Name: "example.com", Type: "CAA", Content: `0 issue "letsencrypt.org"`, TTL: 300}, "@", `0 issue "letsencrypt.org"`},
		{record{ID: 7, Name: "t.example.com", Type: "TXT", Content: "v=spf1 -all", TTL: 300}, "t", `"v=spf1 -all"`},
		// DNSControl joins the strings of a TXT record (and splits again at 255 bytes), so two strings read as one
		{record{ID: 8, Name: "x.example.com", Type: "TXT", Content: `a" "b`, TTL: 300}, "x", `"ab"`},
	}
	for _, c := range cases {
		rc, err := toRecordConfig(dc, c.r)
		if err != nil {
			t.Fatalf("%s %s: %v", c.r.Name, c.r.Type, err)
		}
		if rc.GetLabel() != c.wantLabel {
			t.Errorf("%s %s: label %q, want %q", c.r.Name, c.r.Type, rc.GetLabel(), c.wantLabel)
		}
		if got := rc.GetRDATA().String(); got != c.wantRDATA {
			t.Errorf("%s %s: rdata %q, want %q", c.r.Name, c.r.Type, got, c.wantRDATA)
		}
		if orig, ok := rc.Original.(record); !ok || orig.ID != c.r.ID {
			t.Errorf("%s %s: the original record (and its id) was not kept", c.r.Name, c.r.Type)
		}
	}
}

func TestFromRecordConfig(t *testing.T) {
	dc := newDC(t)
	// read a record from the API, write it back: the request must carry what the API stores
	cases := []struct {
		r        record
		wantType string
		want     string
		wantPrio *int
	}{
		{record{Name: "www.example.com", Type: "CNAME", Content: "example.com", TTL: 300}, "CNAME", "example.com", nil},
		{record{Name: "example.com", Type: "MX", Content: "mail.example.com", Prio: new(10), TTL: 300}, "MX", "mail.example.com", new(10)},
		{record{Name: "_sip._tcp.example.com", Type: "SRV", Content: "5 5060 sip.example.com", Prio: new(10), TTL: 300}, "SRV", "5 5060 sip.example.com", new(10)},
		{record{Name: "t.example.com", Type: "TXT", Content: "v=spf1 -all", TTL: 300}, "TXT", `"v=spf1 -all"`, nil},
		{record{Name: "x.example.com", Type: "TXT", Content: `a" "b`, TTL: 300}, "TXT", `"ab"`, nil},
	}
	for _, c := range cases {
		rc, err := toRecordConfig(dc, c.r)
		if err != nil {
			t.Fatalf("%s %s: %v", c.r.Name, c.r.Type, err)
		}
		in := fromRecordConfig(rc)
		if in.Type != c.wantType || in.Content != c.want {
			t.Errorf("%s %s: got %s %q, want %s %q", c.r.Name, c.r.Type, in.Type, in.Content, c.wantType, c.want)
		}
		if (in.Prio == nil) != (c.wantPrio == nil) || (in.Prio != nil && *in.Prio != *c.wantPrio) {
			t.Errorf("%s %s: prio %v, want %v", c.r.Name, c.r.Type, in.Prio, c.wantPrio)
		}
		if in.TTL != 300 {
			t.Errorf("%s %s: ttl %d, want 300", c.r.Name, c.r.Type, in.TTL)
		}
	}
}

func TestStripSvcbQuotes(t *testing.T) {
	cases := map[string]string{
		`1 . alpn="h2,h3"`:               `1 . alpn=h2,h3`,
		`1 svc.example.com. port="8443"`: `1 svc.example.com. port=8443`,
		`1 . alpn="h2" ech="AEn+DQBF"`:   `1 . alpn=h2 ech="AEn+DQBF"`,
		`1 . ech="AEn+DQBF" alpn="h2"`:   `1 . ech="AEn+DQBF" alpn=h2`,
		`0 target.example.com.`:          `0 target.example.com.`,
	}
	for in, want := range cases {
		if got := stripSvcbQuotes(in); got != want {
			t.Errorf("stripSvcbQuotes(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRetryWait(t *testing.T) {
	if got := retryWait("7", 1).Seconds(); got != 7 {
		t.Errorf("Retry-After 7 gave %v s", got)
	}
	if got := retryWait("600", 1).Seconds(); got != 60 {
		t.Errorf("Retry-After 600 gave %v s, want the 60 s cap", got)
	}
	if got := retryWait("", 3).Seconds(); got != 8 {
		t.Errorf("no Retry-After, attempt 3 gave %v s, want 8", got)
	}
}

func TestPTRInReverseZone(t *testing.T) {
	dc, err := models.NewDomainConfig("2.0.192.in-addr.arpa")
	if err != nil {
		t.Fatal(err)
	}
	rc, err := toRecordConfig(dc, record{Name: "5.2.0.192.in-addr.arpa", Type: "PTR", Content: "host.example.com.", TTL: 300})
	if err != nil {
		t.Fatal(err)
	}
	if rc.GetLabel() != "5" {
		t.Errorf("label %q, want 5", rc.GetLabel())
	}
	if in := fromRecordConfig(rc); in.Type != "PTR" || in.Content != "host.example.com." || in.Name != "5" {
		t.Errorf("PTR request = %+v", in)
	}
}

func TestLongTXTIsSplitAtTheSegmentLimit(t *testing.T) {
	dc := newDC(t)
	long := strings.Repeat("a", 300)
	rc, err := toRecordConfig(dc, record{Name: "l.example.com", Type: "TXT", Content: long, TTL: 300})
	if err != nil {
		t.Fatal(err)
	}
	in := fromRecordConfig(rc)
	want := `"` + strings.Repeat("a", 255) + `" "` + strings.Repeat("a", 45) + `"`
	if in.Content != want {
		t.Errorf("a 300 byte TXT is sent as %q, want two strings (255 + 45)", in.Content)
	}
}

func TestEmptyZoneIsReportedAsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"data":{"records":[],"next_cursor":null}}`))
	}))
	defer srv.Close()
	p := &primednsProvider{client: srv.Client(), apiKey: "k", baseURL: srv.URL}
	dc := newDC(t)
	if _, err := p.GetZoneRecords(dc); !errors.Is(err, errZoneNotFound) {
		t.Errorf("err = %v, want errZoneNotFound", err)
	}
}

func TestApexNSMustUseTheSystemTTL(t *testing.T) {
	dc := newDC(t)
	apex300, _ := dc.NewRecordConfig("@", 300, "NS", "ns1.other.example.")
	apex8000, _ := dc.NewRecordConfig("@", 8000, "NS", "ns1.other.example.")
	sub300, _ := dc.NewRecordConfig("sub", 300, "NS", "ns1.other.example.")
	p := &primednsProvider{}
	errs := p.AuditRecords(models.Records{apex300, apex8000, sub300})
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "TTL(8000)") {
		t.Errorf("errors = %v, want exactly one, for the apex NS with TTL 300", errs)
	}
}

func TestListZonesWithoutTheDomainsScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/domains":
			w.WriteHeader(403)
			w.Write([]byte(`{"success":false,"error":{"code":"INSUFFICIENT_SCOPE","message":"This endpoint requires the 'domains:read' scope."}}`))
		case "/dns/records":
			w.Write([]byte(`{"success":true,"data":{"records":[` +
				`{"id":1,"domain_id":7,"domain_name":"a.example","name":"a.example","type":"SOA"},` +
				`{"id":2,"domain_id":7,"domain_name":"a.example","name":"www.a.example","type":"A"},` +
				`{"id":3,"domain_id":9,"domain_name":"b.example","name":"b.example","type":"SOA"}],"next_cursor":null}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	p := &primednsProvider{client: srv.Client(), apiKey: "k", baseURL: srv.URL}
	names, err := p.ListZones()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "a.example" || names[1] != "b.example" {
		t.Errorf("zones = %v, want [a.example b.example]", names)
	}
}

func TestListZonesOtherErrorsAreReturned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	defer srv.Close()
	p := &primednsProvider{client: srv.Client(), apiKey: "k", baseURL: srv.URL}
	if _, err := p.ListZones(); err == nil {
		t.Error("a 401 must not fall back to the records")
	}
}

func TestSRVNullTargetKeepsItsDot(t *testing.T) {
	dc := newDC(t)
	rc, err := toRecordConfig(dc, record{ID: 1, Name: "_x._tcp.example.com", Type: "SRV", Content: "65 75 .", Prio: new(15), TTL: 300})
	if err != nil {
		t.Fatal(err)
	}
	in := fromRecordConfig(rc)
	if in.Content != "65 75 ." || in.Prio == nil || *in.Prio != 15 {
		t.Errorf("request = %+v, want content %q and prio 15", in, "65 75 .")
	}
}
