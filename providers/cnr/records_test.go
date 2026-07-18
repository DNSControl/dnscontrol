package cnr

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
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

func TestCreateRecordString(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	tests := []struct {
		name    string
		typeAny any
		data    string
		want    string
	}{
		{name: "A", typeAny: dnsv2.TypeA, data: "192.0.2.1", want: "www 300 IN A 192.0.2.1"},
		{name: "AAAA", typeAny: dnsv2.TypeAAAA, data: "2001:db8::1", want: "www 300 IN AAAA 2001:db8::1"},
		{name: "CNAME", typeAny: dnsv2.TypeCNAME, data: "target.example.com.", want: "www 300 IN CNAME target.example.com."},
		{name: "DHCID", typeAny: dnsv2.TypeDHCID, data: "dhcid.example.com.", want: "www 300 IN DHCID dhcid.example.com."},
		{name: "DNAME", typeAny: dnsv2.TypeDNAME, data: "example.net.", want: "www 300 IN DNAME example.net."},
		{name: "MX", typeAny: dnsv2.TypeMX, data: "10 mail.example.net.", want: "www 300 IN MX 10 mail.example.net."},
		{name: "NS", typeAny: dnsv2.TypeNS, data: "ns1.example.net.", want: "www 300 NS ns1.example.net."},
		{name: "PTR", typeAny: dnsv2.TypePTR, data: "192.0.2.1.", want: "www 300 IN PTR 192.0.2.1."},
		{name: "LOC_", typeAny: dnsv2.TypeLOC, data: "52 14 5.000 N 000 08 50.000 E 10.00m 0.00m 0.00m 0.00m", want: "www 300 IN LOC 52 14 5.000 N 00 08 50.000 E 10m 0.00m 0.00m 0.00m"},
		{name: "SSHFP", typeAny: dnsv2.TypeSSHFP, data: "1 2 deadbeef", want: "www 300 IN SSHFP 1 2 DEADBEEF"},
		{name: "NAPTR", typeAny: dnsv2.TypeNAPTR, data: `10 20 "U" "E2U+sip" "!^.*$!sip:customer@example.com!" example.com.`, want: "www 300 IN NAPTR 10 20 \"U\" \"E2U+sip\" \"!^.*$!sip:customer@example.com!\" example.com."},
		{name: "TLSA", typeAny: dnsv2.TypeTLSA, data: "3 1 1 target", want: "www 300 IN TLSA 3 1 1 TARGET"},
		{name: "SMIMEA", typeAny: dnsv2.TypeSMIMEA, data: "3 1 1 target", want: "www 300 IN SMIMEA 3 1 1 target"},
		{name: "CAA", typeAny: dnsv2.TypeCAA, data: `0 issue "letsencrypt.org"`, want: "www 300 IN CAA 0 issue \"letsencrypt.org\""},
		{name: "TXT", typeAny: dnsv2.TypeTXT, data: "hello", want: "www 300 IN TXT " + txtutil.EncodeQuoted("hello")},
		{name: "SRV", typeAny: dnsv2.TypeSRV, data: "5 6 443 service.example.net.", want: "www 300 IN SRV 5 6 443 service.example.net."},
		{name: "SVCB", typeAny: dnsv2.TypeSVCB, data: "1 test.com. port=80", want: "www 300 IN SVCB 1 test.com. port=80"},
		{name: "HTTPS", typeAny: dnsv2.TypeHTTPS, data: "3 test.com. alpn=h2", want: "www 300 IN HTTPS 3 test.com. alpn=h2"},
		// pseudo
		{name: "ANAME", typeAny: "ANAME", data: "example.com.", want: "www 300 IN ANAME example.com."},
		// private
		{name: "ALIAS", typeAny: privatetypes.TypeALIAS, data: "alias.example.com.", want: "www 300 IN ALIAS alias.example.com."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := func() *models.RecordConfig {
				switch tt.typeAny {
				case "ANAME":
					rc := dc.MustNewRecordConfigParse("www", 300, "CNAME", tt.data)
					rc.Type = "ANAME"
					return rc
				default:
					return dc.MustNewRecordConfigParse("www", 300, tt.typeAny, tt.data)
				}
			}()
			got, err := (&Client{}).createRecordString(rc, dc.Name)
			if err != nil {
				t.Fatalf("createRecordString(%s) error = %v", rc.Type, err)
			}
			if got != tt.want {
				t.Fatalf("createRecordString(%s) = %q, want %q", rc.Type, got, tt.want)
			}
		})
	}
}
