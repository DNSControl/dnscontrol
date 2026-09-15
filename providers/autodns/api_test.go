package autodns

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/providers/bind"
)

// TestUpdateZoneClearsMainRecord asserts that the legacy "main" field is not
// echoed back in the zone update. AutoDNS returns the apex address in "main"
// for older zones, and toRecordConfigs() surfaces it as a synthetic apex A
// record. If updateZone() sends it back unchanged, any correction touching the
// apex is undone by the same request that is supposed to apply it: the record
// is removed from resourceRecords and immediately restored from "main", so the
// zone reports the same pending change on every run and can never converge.
func TestUpdateZoneClearsMainRecord(t *testing.T) {
	t.Parallel()

	const domain = "example.com"

	var putBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/zone/_search"):
			_ = json.NewEncoder(w).Encode(JSONResponseDataZone{
				Data: []*Zone{{SystemNameServer: "ns.example.net"}},
			})

		case r.Method == http.MethodGet:
			// A zone that still carries the legacy apex address in "main".
			_ = json.NewEncoder(w).Encode(JSONResponseDataZone{
				Data: []*Zone{{
					Origin:     domain,
					Soa:        &bind.SoaDefaults{TTL: 3600},
					MainRecord: &MainAddressRecord{TTL: 86400, Value: "192.0.2.1"},
					ResourceRecords: []*ResourceRecord{
						{Name: "", Type: "A", Value: "192.0.2.1", TTL: 3600},
					},
				}},
			})

		case r.Method == http.MethodPut:
			putBody, _ = io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(JSONResponseDataZone{Data: []*Zone{{}}})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	api := &autoDNSProvider{baseURL: *baseURL, defaultHeaders: http.Header{}}

	resourceRecords := []*ResourceRecord{
		{Name: "", Type: "A", Value: "192.0.2.1", TTL: 3600},
	}
	nameServers := []*models.Nameserver{{Name: "ns.example.net"}}

	if err := api.updateZone(domain, resourceRecords, nameServers, 3600); err != nil {
		t.Fatalf("updateZone() error = %v", err)
	}

	if len(putBody) == 0 {
		t.Fatal("updateZone() sent no PUT request")
	}

	var sent map[string]any
	if err := json.Unmarshal(putBody, &sent); err != nil {
		t.Fatalf("unmarshal PUT body: %v", err)
	}

	if main, ok := sent["main"]; ok {
		t.Errorf("PUT body still carries main = %v, want the field to be absent", main)
	}

	// The apex must still be described by resourceRecords, otherwise clearing
	// "main" would drop the record instead of relocating it.
	records, ok := sent["resourceRecords"].([]any)
	if !ok {
		t.Fatalf("PUT body has no resourceRecords, got %T", sent["resourceRecords"])
	}
	if len(records) != 1 {
		t.Errorf("PUT body has %d resourceRecords, want 1", len(records))
	}
}
