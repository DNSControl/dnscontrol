package cloudflare

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/zonecache"
	cloudflare "github.com/DNSControl/dnscontrol/v5/providers/cloudflare/cfapi"
)

// dnsRecordUpdateAPI records the body of each DNS record update.
type dnsRecordUpdateAPI struct {
	updates []string
}

func (a *dnsRecordUpdateAPI) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method != http.MethodPut || r.URL.Path != "/zones/zone-id/dns_records/record-id" {
		return nil, fmt.Errorf("unexpected request: %s %s", r.Method, r.URL)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	a.updates = append(a.updates, string(body))
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"success":true,"result":{}}`)), Request: r}, nil
}

// Cloudflare reports some records (e.g. A 1.1.1.1) as not proxiable. A
// correction that turns the proxy on must still ask for it, so that
// Cloudflare rejects it, rather than send proxied=false and report success
// for a change that is planned again on every push (#4776).
func TestModifyProxyOnUnproxiableRecord(t *testing.T) {
	api := &dnsRecordUpdateAPI{}
	client, err := cloudflare.NewWithAPIToken("synthetic-token", cloudflare.BaseURL("https://cloudflare.invalid"), cloudflare.HTTPClient(&http.Client{Transport: api}), cloudflare.UsingRetryPolicy(0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	c := &cloudflareProvider{cfClient: client}
	c.zoneCache = zonecache.New(func() (map[string]cloudflare.Zone, error) {
		return map[string]cloudflare.Zone{"example.com": {ID: "zone-id", Name: "example.com"}}, nil
	})

	dc := models.MustNewDomainConfig("example.com")
	rec := dc.MustNewRecordConfig("foo", 1, "A", "1.1.1.1")
	rec.Metadata[metaProxy] = "on"
	dc.Records = append(dc.Records, rec)

	proxied := false
	existing, err := c.nativeToRecord(dc, cloudflare.DNSRecord{ID: "record-id", Type: "A", Name: "foo.example.com", Content: "1.1.1.1", TTL: 1, Proxied: &proxied, Proxiable: false})
	if err != nil {
		t.Fatal(err)
	}

	corrections, _, err := c.GetZoneRecordsCorrections(dc, models.Records{existing})
	if err != nil {
		t.Fatal(err)
	}
	if len(corrections) != 1 {
		t.Fatalf("got %d corrections, want 1: %v", len(corrections), corrections)
	}
	if err := corrections[0].F(); err != nil {
		t.Fatal(err)
	}
	if len(api.updates) != 1 {
		t.Fatalf("got %d updates, want 1", len(api.updates))
	}
	var sent struct {
		Proxied *bool `json:"proxied"`
	}
	if err := json.Unmarshal([]byte(api.updates[0]), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Proxied == nil || !*sent.Proxied {
		t.Errorf("update must ask for proxied=true, sent %s", api.updates[0])
	}
}
