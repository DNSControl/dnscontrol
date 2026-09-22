package mittwald

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	mwdns "github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/dnsv2"
)

// zoneJSON is a root zone as the API returns it: the A set belongs to an
// ingress, the MX set to mittwald's mail service, the other sets are unset
// except TXT.
const zoneJSON = `{
  "domain": "example.com",
  "id": "00000000-0000-0000-0000-000000000001",
  "recordSet": {
    "mx": {"managed": true},
    "combinedARecords": {"managedBy": {"ingressId": "00000000-0000-0000-0000-000000000002"}},
    "cname": {},
    "txt": {"settings": {"ttl": {"auto": true}}, "entries": ["v=spf1 include:agenturserver.de ~all"]},
    "srv": {},
    "caa": {}
  }
}`

func TestToRCSkipsUnsetAndManagedSets(t *testing.T) {
	var z mwdns.Zone
	if err := json.Unmarshal([]byte(zoneJSON), &z); err != nil {
		t.Fatal(err)
	}
	dc := newDC(t)
	recs, err := toRC(dc, z)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].TypeNum != dnsv2.TypeTXT || recs[0].TTL != autoTTL {
		t.Fatalf("want the TXT record with TTL %d only, got %v", autoTTL, recs)
	}
	if !hasManagedSet(z) {
		t.Error("the ingress and mail sets are managed")
	}
}

func TestToRCSubZone(t *testing.T) {
	var z mwdns.Zone
	if err := json.Unmarshal([]byte(`{"domain": "_autodiscover._tcp.example.com", "id": "00000000-0000-0000-0000-000000000003",
	  "recordSet": {"mx": {}, "combinedARecords": {}, "cname": {}, "txt": {}, "caa": {},
	  "srv": {"settings": {"ttl": {"seconds": 300}}, "records": [{"port": 443, "fqdn": "autodiscover.example.net"}]}}}`), &z); err != nil {
		t.Fatal(err)
	}
	recs, err := toRC(newDC(t), z)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("want one SRV record, got %v", recs)
	}
	rc := recs[0]
	srv := rc.AsSRV()
	if rc.Name != "_autodiscover._tcp" || rc.TTL != 300 || srv.Port != 443 || srv.Target != "autodiscover.example.net." {
		t.Errorf("got %s %d %+v", rc.Name, rc.TTL, srv)
	}
	if hasManagedSet(z) {
		t.Error("an unset set is not managed")
	}
}

func newDC(t *testing.T) *models.DomainConfig {
	t.Helper()
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	return dc
}

func mustRecord(t *testing.T, dc *models.DomainConfig, label string, ttl uint32, typeNum uint16, args ...any) *models.RecordConfig {
	t.Helper()
	rc, err := dc.NewRecordConfig(label, ttl, typeNum, args...)
	if err != nil {
		t.Fatal(err)
	}
	return rc
}

func TestToNativeGroupsAAndAAAA(t *testing.T) {
	dc := newDC(t)
	bodies, err := toNative(models.Records{
		mustRecord(t, dc, "www", 300, dnsv2.TypeA, "192.0.2.1"),
		mustRecord(t, dc, "www", 300, dnsv2.TypeAAAA, "2001:db8::1"),
		mustRecord(t, dc, "www", 300, dnsv2.TypeA, "192.0.2.2"),
	})
	if err != nil {
		t.Fatal(err)
	}
	a := bodies[setA].AlternativeCombinedACustom
	if len(bodies) != 1 || a == nil || len(a.A) != 2 || len(a.Aaaa) != 1 || a.Settings.Ttl.AlternativeTtlSeconds.Seconds != 300 {
		t.Fatalf("got %+v", bodies)
	}
}

func TestToNativeStripsTrailingDot(t *testing.T) {
	dc := newDC(t)
	bodies, err := toNative(models.Records{mustRecord(t, dc, "autoconfig", 300, dnsv2.TypeCNAME, "autoconfig.example.net.")})
	if err != nil {
		t.Fatal(err)
	}
	if got := bodies[setCNAME].AlternativeRecordCNAMEComponent.Fqdn; got != "autoconfig.example.net" {
		t.Errorf("got %q", got)
	}
}

func TestToNativeRejectsTwoTTLsInOneSet(t *testing.T) {
	dc := newDC(t)
	_, err := toNative(models.Records{
		mustRecord(t, dc, "www", 300, dnsv2.TypeA, "192.0.2.1"),
		mustRecord(t, dc, "www", 600, dnsv2.TypeAAAA, "2001:db8::1"),
	})
	if err == nil {
		t.Fatal("A and AAAA share one set and one TTL")
	}
}

func TestPrepDesiredRecordsClampsAndSharesTheATTL(t *testing.T) {
	dc := newDC(t)
	low := mustRecord(t, dc, "low", 30, dnsv2.TypeTXT, "x")
	high := mustRecord(t, dc, "high", 172800, dnsv2.TypeTXT, "x")
	a := mustRecord(t, dc, "www", 600, dnsv2.TypeA, "192.0.2.1")
	aaaa := mustRecord(t, dc, "www", 300, dnsv2.TypeAAAA, "2001:db8::1")
	other := mustRecord(t, dc, "other", 900, dnsv2.TypeA, "192.0.2.2")
	dc.Records = models.Records{low, high, a, aaaa, other}
	prepDesiredRecords(dc)
	for _, c := range []struct {
		rc   *models.RecordConfig
		want uint32
	}{{low, 60}, {high, 86400}, {a, 300}, {aaaa, 300}, {other, 900}} {
		if c.rc.TTL != c.want {
			t.Errorf("%s %s: TTL %d, want %d", c.rc.Name, c.rc.Type, c.rc.TTL, c.want)
		}
	}
}

func TestAuditRecords(t *testing.T) {
	dc := newDC(t)
	for _, tc := range []struct {
		name string
		rc   *models.RecordConfig
		ok   bool
	}{
		{"plain A", mustRecord(t, dc, "www", 300, dnsv2.TypeA, "192.0.2.1"), true},
		{"underscore label", mustRecord(t, dc, "_dmarc", 300, dnsv2.TypeTXT, "v=DMARC1; p=none"), true},
		{"wildcard", mustRecord(t, dc, "*", 300, dnsv2.TypeA, "192.0.2.1"), false},
		{"MX preference above 100", mustRecord(t, dc, "@", 300, dnsv2.TypeMX, 200, "mx.example.net."), false},
		{"null MX", mustRecord(t, dc, "@", 300, dnsv2.TypeMX, 0, "."), false},
		{"TXT with trailing space", mustRecord(t, dc, "t", 300, dnsv2.TypeTXT, "x "), false},
		{"CAA tag", mustRecord(t, dc, "@", 300, dnsv2.TypeCAA, 0, "contactemail", "a@example.com"), false},
		{"CAA issuer", mustRecord(t, dc, "@", 300, dnsv2.TypeCAA, 0, "issue", "letsencrypt.org"), true},
		{"CAA parameters", mustRecord(t, dc, "@", 300, dnsv2.TypeCAA, 0, "issue", "letsencrypt.org; validationmethods=dns-01"), false},
		{"CAA forbid all", mustRecord(t, dc, "@", 300, dnsv2.TypeCAA, 0, "issue", ";"), false},
		{"TXT with one double quote", mustRecord(t, dc, "t", 300, dnsv2.TypeTXT, `"left`), true},
		{"NS", mustRecord(t, dc, "sub", 300, dnsv2.TypeNS, "ns.example.net."), false},
	} {
		if got := len(AuditRecords(models.Records{tc.rc})) == 0; got != tc.ok {
			t.Errorf("%s: accepted=%v, want %v", tc.name, got, tc.ok)
		}
	}

	var many models.Records
	for i := range 11 {
		many = append(many, mustRecord(t, dc, "www", 300, dnsv2.TypeA, fmt.Sprintf("192.0.2.%d", i+1)))
	}
	if errs := AuditRecords(many); len(errs) != 1 {
		t.Errorf("11 A records at one name: want one error, got %v", errs)
	}
}

type fakeRunner struct {
	responses []*http.Response
	calls     int
	body      []string
}

func (f *fakeRunner) Do(req *http.Request) (*http.Response, error) {
	b := ""
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		b = string(raw)
	}
	f.body = append(f.body, b)
	resp := f.responses[min(f.calls, len(f.responses)-1)]
	f.calls++
	return resp, nil
}

func response(code int, headers ...string) *http.Response {
	return responseWithBody(code, "", headers...)
}

func responseWithBody(code int, body string, headers ...string) *http.Response {
	h := http.Header{}
	for i := 0; i+1 < len(headers); i += 2 {
		h.Set(headers[i], headers[i+1])
	}
	return &http.Response{StatusCode: code, Header: h, Body: io.NopCloser(strings.NewReader(body))}
}

// The bodies of the two kinds of 412, as the API sends them.
const (
	eventNotReachedBody = `{"params":{"traceId":"0"},"message":"lastEventID not reached","type":"FailedPrecondition"}`
	cnameConflictBody   = `{"params":{"traceId":"0"},"message":"zone for domain 'x.example.com' contains active records - unable to set CNAME","type":"VError"}`
)

// testRunner returns a runner on a fake clock that advances when it sleeps.
func testRunner(inner *fakeRunner) (*apiRunner, *time.Duration) {
	var slept time.Duration
	start := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	r := newAPIRunner(inner)
	r.sleep = func(d time.Duration) { slept += d }
	r.now = func() time.Time { return start.Add(slept) }
	return r, &slept
}

func TestAPIRunnerRepeatsRateLimitedRequests(t *testing.T) {
	inner := &fakeRunner{responses: []*http.Response{response(429), response(429), response(204)}}
	r, slept := testRunner(inner)
	req, _ := http.NewRequest(http.MethodPut, "https://api.example/v2/x", strings.NewReader(`{"a":1}`))
	resp, err := r.Do(req)
	if err != nil || resp.StatusCode != 204 {
		t.Fatalf("got %v, %v", resp, err)
	}
	if inner.calls != 3 || *slept != 6*time.Second {
		t.Errorf("calls=%d slept=%s", inner.calls, *slept)
	}
	for i, b := range inner.body {
		if b != `{"a":1}` {
			t.Errorf("attempt %d sent body %q", i, b)
		}
	}
}

func TestAPIRunnerRepeatsOnlyIdempotentRequestsAfterServerErrors(t *testing.T) {
	for method, calls := range map[string]int{http.MethodPut: 2, http.MethodPost: 1} {
		inner := &fakeRunner{responses: []*http.Response{response(500), response(204)}}
		r, _ := testRunner(inner)
		req, _ := http.NewRequest(method, "https://api.example/v2/x", strings.NewReader("{}"))
		if _, err := r.Do(req); err != nil {
			t.Fatal(err)
		}
		if inner.calls != calls {
			t.Errorf("%s: %d calls, want %d", method, inner.calls, calls)
		}
	}
}

func TestAPIRunnerLetsRequestsWaitForTheLastWrite(t *testing.T) {
	inner := &fakeRunner{responses: []*http.Response{
		response(204, "ETag", "event-1"),
		responseWithBody(412, eventNotReachedBody),
		response(200),
	}}
	r, _ := testRunner(inner)
	put, _ := http.NewRequest(http.MethodPut, "https://api.example/v2/x", strings.NewReader("{}"))
	if _, err := r.Do(put); err != nil {
		t.Fatal(err)
	}
	// A write to the zone the first write created waits for it, too.
	next, _ := http.NewRequest(http.MethodPut, "https://api.example/v2/x", strings.NewReader("{}"))
	resp, err := r.Do(next)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("got %v, %v", resp, err)
	}
	if got := next.Header.Get("If-Event-Reached"); got != "event-1" {
		t.Errorf("If-Event-Reached = %q", got)
	}
	if inner.calls != 3 {
		t.Errorf("want the 412 repeated, calls=%d", inner.calls)
	}
}

func TestAPIRunnerRepeatsA403AfterAWriteAFewTimes(t *testing.T) {
	inner := &fakeRunner{responses: []*http.Response{response(201, "ETag", "event-1"), response(403)}}
	r, _ := testRunner(inner)
	post, _ := http.NewRequest(http.MethodPost, "https://api.example/v2/x", strings.NewReader("{}"))
	if _, err := r.Do(post); err != nil {
		t.Fatal(err)
	}
	put, _ := http.NewRequest(http.MethodPut, "https://api.example/v2/x", strings.NewReader("{}"))
	resp, err := r.Do(put)
	if err != nil || resp.StatusCode != 403 {
		t.Fatalf("want the 403 back in the end, got %v, %v", resp, err)
	}
	if inner.calls != 1+1+maxLagRetries {
		t.Errorf("calls=%d, want %d", inner.calls, 2+maxLagRetries)
	}
}

func TestAPIRunnerDoesNotRepeatAConflict(t *testing.T) {
	inner := &fakeRunner{responses: []*http.Response{response(204, "ETag", "event-1"), responseWithBody(412, cnameConflictBody)}}
	r, _ := testRunner(inner)
	for range 2 {
		req, _ := http.NewRequest(http.MethodPut, "https://api.example/v2/x", strings.NewReader("{}"))
		resp, err := r.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode == 412 {
			if body, _ := io.ReadAll(resp.Body); string(body) != cnameConflictBody {
				t.Errorf("the caller gets the body, got %q", body)
			}
		}
	}
	if inner.calls != 2 {
		t.Errorf("a CNAME conflict is final, calls=%d", inner.calls)
	}
}

func TestAPIRunnerGivesUp(t *testing.T) {
	inner := &fakeRunner{responses: []*http.Response{response(429)}}
	r, slept := testRunner(inner)
	req, _ := http.NewRequest(http.MethodGet, "https://api.example/v2/x", nil)
	resp, err := r.Do(req)
	if err != nil || resp.StatusCode != 429 {
		t.Fatalf("want the 429 back, got %v, %v", resp, err)
	}
	if *slept > maxRetryWait {
		t.Errorf("waited %s", *slept)
	}
}

func TestAPIRunnerWaitsForReset(t *testing.T) {
	inner := &fakeRunner{responses: []*http.Response{
		response(200, "X-Ratelimit-Remaining", "0", "X-Ratelimit-Reset", "42"),
		response(200, "X-Ratelimit-Remaining", "2999", "X-Ratelimit-Reset", "600"),
	}}
	r, slept := testRunner(inner)
	for range 3 {
		req, _ := http.NewRequest(http.MethodGet, "https://api.example/v2/x", nil)
		if _, err := r.Do(req); err != nil {
			t.Fatal(err)
		}
	}
	if *slept != 42*time.Second {
		t.Errorf("want one wait of 42s for the reset, slept %s", *slept)
	}
}
