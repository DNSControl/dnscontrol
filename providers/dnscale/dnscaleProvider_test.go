package dnscale

import (
	"reflect"
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
		record     Record
		wantLabel  string
		wantTarget string
	}{
		{"A", Record{Name: "www.example.com.", Type: "A", Content: "192.0.2.1", TTL: 300}, "www", "192.0.2.1"},
		{"apex TXT", Record{Name: "example.com.", Type: "TXT", Content: "raw text", TTL: 300}, "@", `"raw text"`},
		{"MX priority field", Record{Name: "example.com.", Type: "MX", Content: "mail.example.net", TTL: 300, Priority: 10}, "@", "10 mail.example.net."},
		{"MX content priority", Record{Name: "example.com.", Type: "MX", Content: "20 mail.example.net", TTL: 300}, "@", "20 mail.example.net."},
		{"SRV", Record{Name: "_sip._tcp.example.com.", Type: "SRV", Content: "1 2 5060 sip.example.net", TTL: 300}, "_sip._tcp", "1 2 5060 sip.example.net."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := toRecordConfig(dc, tc.record)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetLabel(); got != tc.wantLabel {
				t.Errorf("label = %q, want %q", got, tc.wantLabel)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
			if !reflect.DeepEqual(rc.Original, tc.record) {
				t.Error("original provider record was not retained")
			}
		})
	}
}
