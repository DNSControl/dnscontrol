package autodns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestToRecordConfig(t *testing.T) {
	t.Parallel()

	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name     string
		native   *ResourceRecord
		wantType string
		wantData string
	}{
		{"A", &ResourceRecord{Name: "www", Type: "A", Value: "192.0.2.1", TTL: 300}, "A", "192.0.2.1"},
		{"MX", &ResourceRecord{Name: "www", Type: "MX", Value: "mail.example.net.", Pref: 10, TTL: 300}, "MX", "10 mail.example.net."},
		{"SRV", &ResourceRecord{Name: "_sip._tcp", Type: "SRV", Value: "2 443 service.example.net.", Pref: 1, TTL: 300}, "SRV", "1 2 443 service.example.net."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := toRecordConfig(dc, tc.native)
			if err != nil {
				t.Fatalf("toRecordConfig() error = %v", err)
			}
			if rc.Type != tc.wantType {
				t.Errorf("toRecordConfig() type = %q, want %q", rc.Type, tc.wantType)
			}
			if got := rc.GetRDATA().String(); got != tc.wantData {
				t.Errorf("toRecordConfig() data = %q, want %q", got, tc.wantData)
			}
		})
	}
}
