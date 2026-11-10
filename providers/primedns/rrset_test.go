package primedns

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestToRRsetInput(t *testing.T) {
	dc := newDC(t)
	mx1, _ := dc.NewRecordConfig("@", 300, "MX", uint16(10), "mx1.example.net.")
	mx2, _ := dc.NewRecordConfig("@", 300, "MX", uint16(20), "mx2.example.net.")
	in := toRRsetInput(models.Records{mx1, mx2})
	if in.TTL != 300 || len(in.Records) != 2 {
		t.Fatalf("got %+v", in)
	}
	if in.Records[0].Content != "mx1.example.net" || in.Records[0].Prio == nil || *in.Records[0].Prio != 10 {
		t.Errorf("first value %+v", in.Records[0])
	}
	if in.Records[1].Content != "mx2.example.net" || *in.Records[1].Prio != 20 {
		t.Errorf("second value %+v", in.Records[1])
	}
}

type recorded struct{ method, path, body string }

// rrsetServer answers GET of the apex NS RRset with PrimeDNS's own NS plus one foreign NS, and records the other calls.
func rrsetServer(t *testing.T, got *[]recorded) *primednsProvider {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"success":true,"data":{"ttl":86400,"records":[{"content":"ns1.primedns.net"},{"content":"ns2.primedns.net"},{"content":"ns.other.example"}]}}`))
			return
		}
		*got = append(*got, recorded{r.Method, r.URL.EscapedPath(), string(b)})
		w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	t.Cleanup(srv.Close)
	return &primednsProvider{client: srv.Client(), apiKey: "k", baseURL: srv.URL}
}

func TestPutKeepsPrimeDNSApexNS(t *testing.T) {
	var got []recorded
	p := rrsetServer(t, &got)
	err := p.putApexSafe("example.com", "@", "NS", rrsetInput{TTL: 86400, Records: []rrsetValue{{Content: "ns.mine.example"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].method != "PUT" || got[0].path != "/zones/example.com/rrsets/@/NS" {
		t.Fatalf("calls %+v", got)
	}
	for _, want := range []string{"ns1.primedns.net", "ns2.primedns.net", "ns.mine.example"} {
		if !strings.Contains(got[0].body, want) {
			t.Errorf("body %s lacks %s", got[0].body, want)
		}
	}
	if strings.Contains(got[0].body, "ns.other.example") {
		t.Errorf("a foreign NS that is no longer wanted was kept: %s", got[0].body)
	}
}

func TestDeleteApexNSKeepsPrimeDNSNS(t *testing.T) {
	var got []recorded
	p := rrsetServer(t, &got)
	if err := p.deleteApexSafe("example.com", "@", "NS"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].method != "PUT" || strings.Contains(got[0].body, "ns.other.example") || !strings.Contains(got[0].body, `"ttl":86400`) {
		t.Fatalf("calls %+v", got)
	}
}

func TestDeleteOtherRRsetIsPlainDelete(t *testing.T) {
	var got []recorded
	p := rrsetServer(t, &got)
	if err := p.deleteApexSafe("example.com", "*.sub", "A"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].method != "DELETE" || got[0].path != "/zones/example.com/rrsets/%2A.sub/A" && got[0].path != "/zones/example.com/rrsets/*.sub/A" {
		t.Fatalf("calls %+v", got)
	}
}
