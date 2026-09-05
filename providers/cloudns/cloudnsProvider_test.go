package cloudns

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestToRcUsesV3RecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name string
		in   domainRecord
		want *models.RecordConfig
	}{
		{"MX", domainRecord{Type: "MX", Host: "mail", Target: "mx.example.net", Priority: "10", TTL: "300"}, dc.MustNewRecordConfig("mail", 300, dnsv2.TypeMX, uint16(10), "mx.example.net.")},
		{"SRV", domainRecord{Type: "SRV", Host: "_https._tcp", Target: "service.example.net", Priority: "1", Weight: "2", Port: "443", TTL: "300"}, dc.MustNewRecordConfig("_https._tcp", 300, dnsv2.TypeSRV, uint16(1), uint16(2), uint16(443), "service.example.net.")},
		{"CAA", domainRecord{Type: "CAA", Host: "@", CaaFlag: "0", CaaTag: "issue", CaaValue: "letsencrypt.org", TTL: "300"}, dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(0), "issue", "letsencrypt.org")},
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
				t.Error("toRc() did not preserve the original ClouDNS record")
			}
		})
	}
}

func TestToRcConvertsCloudWRToCloudnsWR(t *testing.T) {
	// ClouDNS API returns "CLOUD_WR" as the type for web redirect records.
	// dnscontrol uses "CLOUDNS_WR" as the custom record type.
	// Verify that toRc maps "CLOUD_WR" -> "CLOUDNS_WR" so that fetched
	// records match desired records and are not destroyed/recreated every push.
	r := &domainRecord{
		ID:     "123",
		Type:   "CLOUD_WR",
		Host:   "www",
		Target: "\"https://example.com\"",
		TTL:    "3600",
	}

	dc := models.MustNewDomainConfig("example.com")
	rc, err := toRc(dc, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.Type != "CLOUDNS_WR" {
		t.Errorf("expected type CLOUDNS_WR, got %s", rc.Type)
	}
	gotTarget := rc.AsCLOUDNSWR().Target
	wantTarget := r.Target
	if gotTarget != wantTarget {
		t.Errorf("wanted target %s, got %s", wantTarget, gotTarget)
	}
}

func TestToRcMX(t *testing.T) {
	r := &domainRecord{
		Type:     "MX",
		Host:     "www",
		Target:   "mail.example.net",
		Priority: "10",
		TTL:      "3600",
	}

	dc := models.MustNewDomainConfig("example.com")
	rc, err := toRc(dc, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := rc.GetRDATA().String(); got != "10 mail.example.net." {
		t.Errorf("expected MX data %q, got %q", "10 mail.example.net.", got)
	}
}
