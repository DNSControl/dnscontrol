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
	"github.com/cloudflare/cloudflare-go"
)

// workerRouteAPI serves a fixed list of Worker Routes through the real
// Cloudflare SDK; no credentials or network access are needed.
type workerRouteAPI struct {
	routes []cloudflare.WorkerRoute
}

func (a *workerRouteAPI) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method != http.MethodGet || r.URL.Path != "/zones/zone-id/workers/routes" {
		return nil, fmt.Errorf("unexpected request: %s %s", r.Method, r.URL)
	}
	data, err := json.Marshal(struct {
		Success bool                     `json:"success"`
		Result  []cloudflare.WorkerRoute `json:"result"`
	}{true, a.routes})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(data))), Request: r}, nil
}

// A Worker Route has no DNS TTL. Provider read-back constructs TTL 1, so a
// desired route that inherited DEFAULTS(DefaultTTL(...)) must not produce a
// TTL-only correction on every preview.
func TestWorkerRouteDefaultTTLNoop(t *testing.T) {
	for _, ttl := range []uint32{1, 300, 86400} {
		t.Run(fmt.Sprintf("ttl=%d", ttl), func(t *testing.T) {
			api := &workerRouteAPI{routes: []cloudflare.WorkerRoute{{ID: "route-id", Pattern: "example.com/*", ScriptName: "my-worker"}}}
			client, err := cloudflare.NewWithAPIToken("synthetic-token", cloudflare.BaseURL("https://cloudflare.invalid"), cloudflare.HTTPClient(&http.Client{Transport: api}), cloudflare.UsingRateLimit(100000), cloudflare.UsingRetryPolicy(0, 0, 0))
			if err != nil {
				t.Fatal(err)
			}
			c := &cloudflareProvider{cfClient: client, manageWorkers: true}
			c.zoneCache = zonecache.New(func() (map[string]cloudflare.Zone, error) {
				return map[string]cloudflare.Zone{"example.com": {ID: "zone-id", Name: "example.com"}}, nil
			})

			dc := models.MustNewDomainConfig("example.com")
			dc.Records = append(dc.Records, dc.MustNewRecordConfig("@", ttl, "CF_WORKER_ROUTE", "example.com/*", "my-worker"))

			existing, err := c.getWorkerRoutes("zone-id", dc)
			if err != nil {
				t.Fatal(err)
			}
			corrections, _, err := c.GetZoneRecordsCorrections(dc, existing)
			if err != nil {
				t.Fatal(err)
			}
			for _, correction := range corrections {
				t.Errorf("unexpected correction: %s", correction.Msg)
			}
		})
	}
}
