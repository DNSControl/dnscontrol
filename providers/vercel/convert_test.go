package vercel

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestVercelRecordToRCUsesV3RecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name string
		in   DNSRecord
		want *models.RecordConfig
	}{
		{"A", DNSRecord{Name: "www", RecordType: "A", Value: "192.0.2.1", TTL: 300, Type: "A"}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1")},
		{"TXT", DNSRecord{Name: "www", RecordType: "TXT", Value: "hello world", TTL: 300, Type: "TXT"}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, "hello world")},
		{"MX", DNSRecord{Name: "www", RecordType: "MX", Value: "mail.example.net", TTL: 300, Type: "MX", MXPriority: 10}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeMX, uint16(10), "mail.example.net.")},
		{"SRV", DNSRecord{Name: "www", RecordType: "SRV", Value: "2 443 service.example.net.", TTL: 300, Priority: 1, Type: "SRV"}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeSRV, uint16(1), uint16(2), uint16(443), "service.example.net.")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vercelRecordToRC(dc, tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if got.NameFQDN != tt.want.NameFQDN || got.TTL != tt.want.TTL || got.TypeNum != tt.want.TypeNum || got.GetRDATA().String() != tt.want.GetRDATA().String() {
				t.Errorf("vercelRecordToRC() = %s %d IN %s %s, want %s %d IN %s %s", got.NameFQDN, got.TTL, got.Type, got.GetRDATA(), tt.want.NameFQDN, tt.want.TTL, tt.want.Type, tt.want.GetRDATA())
			}
			if got.Original != tt.in {
				t.Error("vercelRecordToRC() did not preserve the original Vercel record")
			}
		})
	}
}
