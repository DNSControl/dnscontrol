package digitalocean

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/digitalocean/godo"
)

func TestToRcUsesV3RecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name string
		in   godo.DomainRecord
		want *models.RecordConfig
	}{
		{"TXT", godo.DomainRecord{Name: "www", Type: "TXT", Data: "hello world", TTL: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, "hello world")},
		{"MX", godo.DomainRecord{Name: "www", Type: "MX", Data: "mail.example.net", TTL: 300, Priority: 10}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeMX, uint16(10), "mail.example.net.")},
		{"SRV", godo.DomainRecord{Name: "www", Type: "SRV", Data: "service.example.net", TTL: 300, Priority: 1, Weight: 2, Port: 443}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeSRV, uint16(1), uint16(2), uint16(443), "service.example.net.")},
		{"CAA", godo.DomainRecord{Name: "www", Type: "CAA", Data: "letsencrypt.org", TTL: 300, Flags: 0, Tag: "issue"}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeCAA, uint8(0), "issue", "letsencrypt.org")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toRc(dc, &tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if got.NameFQDN != tt.want.NameFQDN || got.TTL != tt.want.TTL || got.TypeNum != tt.want.TypeNum || got.GetRDATA().String() != tt.want.GetRDATA().String() {
				t.Errorf("toRc() = %s %d IN %s %s, want %s %d IN %s %s", got.NameFQDN, got.TTL, got.Type, got.GetRDATA(), tt.want.NameFQDN, tt.want.TTL, tt.want.Type, tt.want.GetRDATA())
			}
			if got.Original != &tt.in {
				t.Error("toRc() did not preserve the original DigitalOcean record")
			}
		})
	}
}
