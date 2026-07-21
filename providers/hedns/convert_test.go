package hedns

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestRecordToRCUsesV3RecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name string
		in   Record
		want *models.RecordConfig
	}{
		{"A", Record{Name: "www.example.com", Type: "A", Data: "192.0.2.1", TTL: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1")},
		{"MX", Record{Name: "www.example.com", Type: "MX", Data: "mail.example.net", TTL: 300, Priority: 10}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeMX, uint16(10), "mail.example.net.")},
		{"SRV", Record{Name: "www.example.com", Type: "SRV", Data: "2 443 service.example.net.", TTL: 300, Priority: 1}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeSRV, uint16(1), uint16(2), uint16(443), "service.example.net.")},
		{"SPF becomes TXT", Record{Name: "www.example.com", Type: "SPF", Data: `"v=spf1 -all"`, TTL: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, "v=spf1 -all")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := recordToRC(dc, tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if got.NameFQDN != tt.want.NameFQDN || got.TTL != tt.want.TTL || got.TypeNum != tt.want.TypeNum || got.GetRDATA().String() != tt.want.GetRDATA().String() {
				t.Errorf("recordToRC() = %s %d IN %s %s, want %s %d IN %s %s", got.NameFQDN, got.TTL, got.Type, got.GetRDATA(), tt.want.NameFQDN, tt.want.TTL, tt.want.Type, tt.want.GetRDATA())
			}
			if got.Original != tt.in {
				t.Error("recordToRC() did not preserve the original HE.net record")
			}
		})
	}
}
