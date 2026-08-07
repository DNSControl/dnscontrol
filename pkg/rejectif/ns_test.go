package rejectif

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestNsAtApex(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		wantErr bool
	}{
		{name: "apex as @", label: "@", wantErr: true},
		{name: "subdomain", label: "xyz", wantErr: false},
		{name: "deep subdomain", label: "a.b.c", wantErr: false},
	}

	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("placeholder", 300, dnsv2.TypeNS, "ns1.example.net.")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc.Name = tt.label
			if err := NsAtApex(rc); (err != nil) != tt.wantErr {
				t.Errorf("NsAtApex(label=%q) error = %v, wantErr %v", tt.label, err, tt.wantErr)
			}
		})
	}
}
