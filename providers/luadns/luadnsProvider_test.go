package luadns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	api "github.com/luadns/luadns-go"
)

func TestNativeToRecord(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		record     api.Record
		wantTarget string
	}{
		{"A", api.Record{Name: "www.example.com.", Type: "A", Content: "192.0.2.1", TTL: 300}, "192.0.2.1"},
		{"MX", api.Record{Name: "example.com.", Type: "MX", Content: "10 mail.example.net.", TTL: 300}, "10 mail.example.net."},
		{"TXT", api.Record{Name: "example.com.", Type: "TXT", Content: "raw text", TTL: 300}, `"raw text"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.record
			rc, err := nativeToRecord(dc, &record)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
		})
	}
}

// TestRecordsToNativeHTTPS verifies that HTTPS records are serialized with
// unquoted SvcParams (port=80, not port="80"), which LuaDNS's strict parser
// requires.
func TestRecordsToNativeHTTPS(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		priority    uint16
		target      string
		params      string
		wantContent string
	}{
		{"port", 1, "example.com.", "port=80", "1 example.com. port=80"},
		{"alpn+port", 3, "example.com.", "alpn=h2,h3 port=999", "3 example.com. alpn=h2,h3 port=999"},
		{"alias-mode-no-params", 0, "example.com.", "", "0 example.com."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := dc.NewRecordConfig("@", 300, "HTTPS", tc.priority, tc.target, tc.params)
			if err != nil {
				t.Fatal(err)
			}
			rrs := recordsToNative([]*models.RecordConfig{rc})
			if len(rrs) != 1 {
				t.Fatalf("recordsToNative returned %d records, want 1", len(rrs))
			}
			if got := rrs[0].Content; got != tc.wantContent {
				t.Errorf("content = %q, want %q", got, tc.wantContent)
			}
		})
	}
}
