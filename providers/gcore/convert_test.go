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
		rrname     string
		rrtype     string
		content    []any
		wantLabel  string
		wantFQDN   string
		wantTarget string
	}{
		{"A", "www.example.com", "A", []any{"192.0.2.1"}, "www", "www.example.com", "192.0.2.1"},
		{"apex A", "example.com", "A", []any{"192.0.2.1"}, "@", "example.com", "192.0.2.1"},
		{"nested AAAA", "ygg.irc.example.com", "AAAA", []any{"2001:db8::1"}, "ygg.irc", "ygg.irc.example.com", "2001:db8::1"},
		{"MX", "www.example.com", "MX", []any{int64(10), "mail.example.net."}, "www", "www.example.com", "10 mail.example.net."},
		{"CAA", "example.com", "CAA", []any{int64(0), "issue", "letsencrypt.org"}, "@", "example.com", `0 issue "letsencrypt.org"`},
		{"TXT", "www.example.com", "TXT", []any{"raw text"}, "www", "www.example.com", `"raw text"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			records, err := nativeToRecords(gcoreRRSetExtended{
				Name: tc.rrname, Type: tc.rrtype, TTL: 300,
				Records: []dnssdk.ResourceRecord{{Content: tc.content}},
			}, dc)
			if err != nil {
				t.Fatal(err)
			}
			if got := records[0].GetLabel(); got != tc.wantLabel {
				t.Errorf("label = %q, want %q", got, tc.wantLabel)
			}
			if got := records[0].GetLabelFQDN(); got != tc.wantFQDN {
				t.Errorf("fqdn = %q, want %q", got, tc.wantFQDN)
			}
			if got := records[0].GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
		})
	}
}
