package dynu

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestToRc(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}

	priority := 10
	zero := 0
	weight := 2
	port := 5060
	tests := []struct {
		name       string
		record     dynuRecord
		wantTarget string
	}{
		{"A", dynuRecord{NodeName: "www", RecordType: "A", IPv4Address: "192.0.2.1", TTL: 300}, "192.0.2.1"},
		{"MX", dynuRecord{NodeName: "@", RecordType: "MX", Host: "mail.example.net", Priority: &priority, TTL: 300}, "10 mail.example.net."},
		{"null MX", dynuRecord{NodeName: "@", RecordType: "MX", Host: "example.com", Priority: &zero, TTL: 300}, "0 ."},
		{"TXT", dynuRecord{NodeName: "@", RecordType: "TXT", TextData: "raw text", TTL: 300}, `"raw text"`},
		{"SRV", dynuRecord{NodeName: "_sip._tcp", RecordType: "SRV", Host: "sip.example.net", Priority: &priority, Weight: &weight, Port: &port, TTL: 300}, "10 2 5060 sip.example.net."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.record
			rc, err := toRc(&record, dc)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
			if rc.Original != &record {
				t.Error("original provider record was not retained")
			}
		})
	}
}
