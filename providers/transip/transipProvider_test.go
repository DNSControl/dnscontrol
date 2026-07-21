package transip

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/transip/gotransip/v6/domain"
)

func TestNativeToRecordUsesV3RecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name string
		in   domain.DNSEntry
		want *models.RecordConfig
	}{
		{"A", domain.DNSEntry{Name: "www", Type: "A", Content: "192.0.2.1", Expire: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1")},
		{"TXT quoted", domain.DNSEntry{Name: "www", Type: "TXT", Content: `"hello world"`, Expire: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, "hello world")},
		{"MX", domain.DNSEntry{Name: "www", Type: "MX", Content: "10 mail.example.net.", Expire: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeMX, uint16(10), "mail.example.net.")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nativeToRecord(tt.in, dc)
			if err != nil {
				t.Fatal(err)
			}
			if got.NameFQDN != tt.want.NameFQDN || got.TTL != tt.want.TTL || got.TypeNum != tt.want.TypeNum || got.GetRDATA().String() != tt.want.GetRDATA().String() {
				t.Errorf("nativeToRecord() = %s %d IN %s %s, want %s %d IN %s %s", got.NameFQDN, got.TTL, got.Type, got.GetRDATA(), tt.want.NameFQDN, tt.want.TTL, tt.want.Type, tt.want.GetRDATA())
			}
			if got.Original != tt.in {
				t.Error("nativeToRecord() did not preserve the original TransIP record")
			}
		})
	}
}
