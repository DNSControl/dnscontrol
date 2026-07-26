package dnsimple

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	dnsimpleapi "github.com/dnsimple/dnsimple-go/v8/dnsimple"
)

func TestToRecordConfig(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		record     dnsimpleapi.ZoneRecord
		wantLabel  string
		wantTarget string
	}{
		{"A", dnsimpleapi.ZoneRecord{Name: "www", Type: "A", Content: "192.0.2.1", TTL: 300}, "www", "192.0.2.1"},
		{"apex TXT", dnsimpleapi.ZoneRecord{Name: "", Type: "TXT", Content: `"quoted text"`, TTL: 300}, "@", `"quoted text"`},
		{"MX", dnsimpleapi.ZoneRecord{Name: "", Type: "MX", Content: "mail.example.net", TTL: 300, Priority: 10}, "@", "10 mail.example.net."},
		{"SRV", dnsimpleapi.ZoneRecord{Name: "_sip._tcp", Type: "SRV", Content: "2 5060 sip.example.net.", TTL: 300, Priority: 1}, "_sip._tcp", "1 2 5060 sip.example.net."},
		{"HTTPS alias mode", dnsimpleapi.ZoneRecord{Name: "", Type: "HTTPS", Content: "0 target.example.net", TTL: 300}, "@", "0 target.example.net."},
		{"HTTPS with params", dnsimpleapi.ZoneRecord{Name: "", Type: "HTTPS", Content: `3 target.example.net alpn="h2,h3" port="999"`, TTL: 300}, "@", `3 target.example.net. alpn="h2,h3" port="999"`},
		{"SVCB alias mode", dnsimpleapi.ZoneRecord{Name: "", Type: "SVCB", Content: "0 target.example.net", TTL: 300}, "@", "0 target.example.net."},
		{"SVCB already qualified", dnsimpleapi.ZoneRecord{Name: "", Type: "SVCB", Content: "0 target.example.net.", TTL: 300}, "@", "0 target.example.net."},
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
			if got := rc.GetTargetCombined(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
			original, ok := rc.Original.(dnsimpleapi.ZoneRecord)
			if !ok || original.Type != tc.record.Type {
				t.Error("original provider record was not retained")
			}
		})
	}
}
