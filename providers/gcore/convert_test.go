package gcore

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	dnssdk "github.com/G-Core/gcore-dns-sdk-go"
)

func TestNativeToRecords(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		rrtype     string
		content    []any
		wantTarget string
	}{
		{"A", "A", []any{"192.0.2.1"}, "192.0.2.1"},
		{"MX", "MX", []any{int64(10), "mail.example.net."}, "10 mail.example.net."},
		{"CAA", "CAA", []any{int64(0), "issue", "letsencrypt.org"}, `0 issue "letsencrypt.org"`},
		{"TXT", "TXT", []any{"raw text"}, `"raw text"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			records, err := nativeToRecords(gcoreRRSetExtended{
				Name: "www.example.com.", Type: tc.rrtype, TTL: 300,
				Records: []dnssdk.ResourceRecord{{Content: tc.content}},
			}, dc)
			if err != nil {
				t.Fatal(err)
			}
			if got := records[0].GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
		})
	}
}
