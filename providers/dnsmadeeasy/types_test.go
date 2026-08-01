package dnsmadeeasy

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
		record     recordResponseDataEntry
		wantType   string
		wantTarget string
	}{
		{"MX", recordResponseDataEntry{Name: "@", Type: "MX", Value: "mail.example.net.", TTL: 300, MxLevel: 10}, "MX", "10 mail.example.net."},
		{"SRV", recordResponseDataEntry{Name: "_sip._tcp", Type: "SRV", Value: "sip.example.net.", TTL: 300, Priority: 1, Weight: 2, Port: 5060}, "SRV", "1 2 5060 sip.example.net."},
		{"CAA", recordResponseDataEntry{Name: "@", Type: "CAA", Value: `"letsencrypt.org"`, TTL: 300, CaaType: "issue", IssuerCritical: 1}, "CAA", `1 issue "letsencrypt.org"`},
		{"TXT", recordResponseDataEntry{Name: "@", Type: "TXT", Value: `"quoted text"`, TTL: 300}, "TXT", `"quoted text"`},
		{"ANAME", recordResponseDataEntry{Name: "@", Type: "ANAME", Value: "target.example.net.", TTL: 300}, "ALIAS", "target.example.net."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.record
			rc := toRecordConfig(dc, &record)
			if got := rc.Type; got != tc.wantType {
				t.Errorf("type = %q, want %q", got, tc.wantType)
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
