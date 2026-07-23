package infomaniak

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestToRecordConfig(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		record     dnsRecord
		wantTarget string
	}{
		{"A", dnsRecord{Source: "www", Type: "A", TTL: 300, Target: "192.0.2.1"}, "192.0.2.1"},
		{"MX", dnsRecord{Source: "", Type: "MX", TTL: 300, Target: "10 mail.example.net"}, "10 mail.example.net."},
		{"TXT", dnsRecord{Source: ".", Type: "TXT", TTL: 300, Target: `"raw text"`}, `"raw text"`},
		{"SRV", dnsRecord{Source: "_sip._tcp", Type: "SRV", TTL: 300, Target: "1 2 5060 sip.example.net"}, "1 2 5060 sip.example.net."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := toRecordConfig(dc, tc.record)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
		})
	}
}
