package fortigate

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestNativeToRecord(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		record     fgDNSRecord
		wantLabel  string
		wantTarget string
	}{
		{"A", fgDNSRecord{Type: "A", Hostname: "www", IP: "192.0.2.1", TTL: 300, Status: "enable"}, "www", "192.0.2.1"},
		{"A", fgDNSRecord{Type: "A", Hostname: "@", IP: "192.0.2.1", TTL: 300, Status: "enable"}, "@", "192.0.2.1"},
		{"A", fgDNSRecord{Type: "A", Hostname: "example.com.", IP: "192.0.2.1", TTL: 300, Status: "enable"}, "@", "192.0.2.1"},
		{"CNAME", fgDNSRecord{Type: "CNAME", Hostname: "www", CanonicalName: "target.example.net.", TTL: 300, Status: "enable"}, "www", "target.example.net."},
		{"MX", fgDNSRecord{Type: "MX", Hostname: "mail.example.net.", Preference: 10, TTL: 300, Status: "disable"}, "@", "10 mail.example.net."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := nativeToRecord(dc, tc.record)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
			if got := rc.GetLabel(); got != tc.wantLabel {
				t.Errorf("label = %q, want %q", got, tc.wantLabel)
			}
		})
	}
}
