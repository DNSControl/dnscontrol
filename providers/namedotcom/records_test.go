package namedotcom

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/namedotcom/go/namecom"
)

func TestToRecordUsesV3RecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name string
		in   namecom.Record
		want *models.RecordConfig
	}{
		{"A", namecom.Record{Fqdn: "www.example.com.", Type: "A", Answer: "192.0.2.1", TTL: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1")},
		{"TXT", namecom.Record{Fqdn: "www.example.com.", Type: "TXT", Answer: "hello world", TTL: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, "hello world")},
		{"MX", namecom.Record{Fqdn: "www.example.com.", Type: "MX", Answer: "mail.example.net.", TTL: 300, Priority: 10}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeMX, uint16(10), "mail.example.net.")},
		{"SRV", namecom.Record{Fqdn: "www.example.com.", Type: "SRV", Answer: "2 443 service.example.net", TTL: 300, Priority: 1}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeSRV, uint16(1), uint16(2), uint16(443), "service.example.net.")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toRecord(&tt.in, dc)
			if err != nil {
				t.Fatal(err)
			}
			if got.NameFQDN != tt.want.NameFQDN || got.TTL != tt.want.TTL || got.TypeNum != tt.want.TypeNum || got.GetRDATA().String() != tt.want.GetRDATA().String() {
				t.Errorf("toRecord() = %s %d IN %s %s, want %s %d IN %s %s", got.NameFQDN, got.TTL, got.Type, got.GetRDATA(), tt.want.NameFQDN, tt.want.TTL, tt.want.Type, tt.want.GetRDATA())
			}
			if got.Original != &tt.in {
				t.Error("toRecord() did not preserve the original Name.com record")
			}
		})
	}
}
