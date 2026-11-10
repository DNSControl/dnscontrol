package primedns

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// apexServer answers every GET with the given status and body and records the other calls (and counts all calls).
func apexServer(t *testing.T, status int, body string, calls *int, got *[]recorded) *primednsProvider {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		b, _ := io.ReadAll(r.Body)
		if r.Method == http.MethodGet {
			w.WriteHeader(status)
			w.Write([]byte(body))
			return
		}
		*got = append(*got, recorded{r.Method, r.URL.EscapedPath(), string(b)})
		w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	t.Cleanup(srv.Close)
	return &primednsProvider{client: srv.Client(), apiKey: "k", baseURL: srv.URL}
}

const apexAnswer = `{"success":true,"data":{"ttl":8000,"records":[{"content":"ns1.no-ip.pl"},{"content":"ns2.no-ip.pl"},{"content":"ns1.primedns.net"},{"content":"ns.other.example"}]}}`

func TestOwnApexNSIsOnlyAskedForTheApexNSRRset(t *testing.T) {
	for _, c := range []struct{ name, typ string }{{"www", "NS"}, {"@", "A"}, {"@", "MX"}, {"sub", "A"}} {
		calls := 0
		p := apexServer(t, 200, apexAnswer, &calls, new([]recorded))
		own, err := p.ownApexNS("example.com", c.name, c.typ)
		if err != nil || own != nil || calls != 0 {
			t.Errorf("%s %s: own=%v err=%v, %d calls to the API; want nothing", c.name, c.typ, own, err, calls)
		}
	}
}

func TestOwnApexNSKeepsOurNameServersUnderBothNames(t *testing.T) {
	calls := 0
	p := apexServer(t, 200, apexAnswer, &calls, new([]recorded))
	own, err := p.ownApexNS("example.com", "@", "NS")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range own {
		got = append(got, v.Content)
	}
	if strings.Join(got, ",") != "ns1.no-ip.pl,ns2.no-ip.pl,ns1.primedns.net" {
		t.Errorf("own = %v, want PrimeDNS's own name servers in the order the API listed them and not ns.other.example", got)
	}
}

func TestOwnApexNSWithoutAnNSRRsetIsEmptyNotAnError(t *testing.T) {
	calls := 0
	p := apexServer(t, 404, `{"success":false,"error":{"code":"RRSET_NOT_FOUND","message":"RRset not found"}}`, &calls, new([]recorded))
	own, err := p.ownApexNS("example.com", "@", "NS")
	if err != nil || len(own) != 0 {
		t.Errorf("own=%v err=%v, want an empty list and no error for a zone with no apex NS RRset", own, err)
	}
}

func TestOwnApexNSReturnsOtherAPIErrors(t *testing.T) {
	calls := 0
	p := apexServer(t, 403, `{"success":false,"error":{"code":"INSUFFICIENT_SCOPE","message":"no"}}`, &calls, new([]recorded))
	if _, err := p.ownApexNS("example.com", "@", "NS"); err == nil {
		t.Error("a 403 from the API was swallowed: the PUT would have dropped PrimeDNS's own name servers")
	}
}

// If the lookup of the current RRset fails the provider must not send a PUT without PrimeDNS's own records.
func TestPutApexNSDoesNotWriteWhenTheLookupFails(t *testing.T) {
	calls := 0
	var got []recorded
	p := apexServer(t, 403, `{"success":false,"error":{"code":"INSUFFICIENT_SCOPE","message":"no"}}`, &calls, &got)
	if err := p.putApexSafe("example.com", "@", "NS", rrsetInput{TTL: 8000, Records: []rrsetValue{{Content: "ns.mine.example"}}}); err == nil {
		t.Error("putApexSafe returned no error")
	}
	if len(got) != 0 {
		t.Errorf("a write was sent after the failed lookup: %+v", got)
	}
}

// The TTL of the PUT is the one the user wrote (the audit makes it the TTL of PrimeDNS's own records).
func TestPutApexNSSendsTheRequestedTTLAndOurRecordsFirst(t *testing.T) {
	calls := 0
	var got []recorded
	p := apexServer(t, 200, apexAnswer, &calls, &got)
	if err := p.putApexSafe("example.com", "@", "NS", rrsetInput{TTL: 8000, Records: []rrsetValue{{Content: "ns.mine.example"}}}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].body, `"ttl":8000`) {
		t.Fatalf("calls %+v", got)
	}
	body := got[0].body
	if strings.Index(body, "ns1.no-ip.pl") > strings.Index(body, "ns.mine.example") || strings.Contains(body, "ns.other.example") {
		t.Errorf("body %s: our name servers first, the user's after them, the old foreign one gone", body)
	}
}
