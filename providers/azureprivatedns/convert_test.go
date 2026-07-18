package azureprivatedns

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	adns "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestNativeToRecordsUsesV3RecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name       string
		azureType  string
		properties *adns.RecordSetProperties
		want       *models.RecordConfig
	}{
		{"A", "Microsoft.Network/privateDnsZones/A", &adns.RecordSetProperties{ARecords: []*adns.ARecord{{IPv4Address: new("192.0.2.1")}}}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1")},
		{"AAAA", "Microsoft.Network/privateDnsZones/AAAA", &adns.RecordSetProperties{AaaaRecords: []*adns.AaaaRecord{{IPv6Address: new("2001:db8::1")}}}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeAAAA, "2001:db8::1")},
		{"CNAME", "Microsoft.Network/privateDnsZones/CNAME", &adns.RecordSetProperties{CnameRecord: &adns.CnameRecord{Cname: new("alias.example.net.")}}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeCNAME, "alias.example.net.")},
		{"PTR", "Microsoft.Network/privateDnsZones/PTR", &adns.RecordSetProperties{PtrRecords: []*adns.PtrRecord{{Ptrdname: new("host.example.net.")}}}, dc.MustNewRecordConfig("www", 300, dnsv2.TypePTR, "host.example.net.")},
		{"empty TXT", "Microsoft.Network/privateDnsZones/TXT", &adns.RecordSetProperties{}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, "")},
		{"segmented TXT", "Microsoft.Network/privateDnsZones/TXT", &adns.RecordSetProperties{TxtRecords: []*adns.TxtRecord{{Value: []*string{new("first"), new("second")}}}}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, "firstsecond")},
		{"MX", "Microsoft.Network/privateDnsZones/MX", &adns.RecordSetProperties{MxRecords: []*adns.MxRecord{{Preference: new(int32(10)), Exchange: new("mail.example.net.")}}}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeMX, uint16(10), "mail.example.net.")},
		{"SRV", "Microsoft.Network/privateDnsZones/SRV", &adns.RecordSetProperties{SrvRecords: []*adns.SrvRecord{{Priority: new(int32(1)), Weight: new(int32(2)), Port: new(int32(443)), Target: new("service.example.net.")}}}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeSRV, uint16(1), uint16(2), uint16(443), "service.example.net.")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.properties.Fqdn = new("www.example.com.")
			tt.properties.TTL = new(int64(300))
			set := &adns.RecordSet{Type: new(tt.azureType), Properties: tt.properties}
			got := nativeToRecords(set, dc)
			if len(got) != 1 {
				t.Fatalf("nativeToRecords() returned %d records, want 1", len(got))
			}
			if got[0].NameFQDN != tt.want.NameFQDN || got[0].TTL != tt.want.TTL || got[0].TypeNum != tt.want.TypeNum || got[0].GetRDATA().String() != tt.want.GetRDATA().String() {
				t.Errorf("nativeToRecords() = %s %d IN %s %s, want %s %d IN %s %s", got[0].NameFQDN, got[0].TTL, got[0].Type, got[0].GetRDATA(), tt.want.NameFQDN, tt.want.TTL, tt.want.Type, tt.want.GetRDATA())
			}
			if got[0].Original != set {
				t.Error("nativeToRecords() did not preserve the original Azure record set")
			}
		})
	}
}
