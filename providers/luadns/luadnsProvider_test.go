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
