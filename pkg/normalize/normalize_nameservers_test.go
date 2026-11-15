package normalize

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestNormalizeNameservers(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		want     string
		wantErrs int
	}{
		{"short name", "ns1", "ns1.example.com", 0},
		{"fqdn with dot", "ns1.example.net.", "ns1.example.net", 0},
		{"fqdn in zone", "ns1.example.com.", "ns1.example.com", 0},
		{"invalid char", "ns1!", "ns1!.example.com", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := models.MustNewDomainConfig("example.com")
			dc.Nameservers = []*models.Nameserver{{Name: tt.in}}
			errs := normalizeNameservers(dc)
			if len(errs) != tt.wantErrs {
				t.Errorf("normalizeNameservers() returned %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
			if got := dc.Nameservers[0].Name; got != tt.want {
				t.Errorf("normalizeNameservers() name = %q, want %q", got, tt.want)
			}
		})
	}
}
