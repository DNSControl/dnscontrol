package cnr

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestToRecordUsesV3RecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name string
		in   Record
		want *models.RecordConfig
	}{
		{"A", Record{Fqdn: "www.example.com.", Type: "A", Answer: "192.0.2.1", TTL: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1")},
		{"MX", Record{Fqdn: "www.example.com.", Type: "MX", Answer: "mail.example.net.", TTL: 300, Priority: 10}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeMX, uint16(10), "mail.example.net.")},
		{"SRV", Record{Fqdn: "www.example.com.", Type: "SRV", Answer: "1 2 443 service.example.net.", TTL: 300, Priority: 1}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeSRV, uint16(1), uint16(2), uint16(443), "service.example.net.")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toRecord(&tt.in, dc)
			if got.NameFQDN != tt.want.NameFQDN || got.TTL != tt.want.TTL || got.TypeNum != tt.want.TypeNum || got.GetRDATA().String() != tt.want.GetRDATA().String() {
				t.Errorf("toRecord() = %s %d IN %s %s, want %s %d IN %s %s", got.NameFQDN, got.TTL, got.Type, got.GetRDATA(), tt.want.NameFQDN, tt.want.TTL, tt.want.Type, tt.want.GetRDATA())
			}
			if got.Original != &tt.in {
				t.Error("toRecord() did not preserve the original CNR record")
			}
		})
	}
}
