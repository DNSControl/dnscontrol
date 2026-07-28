package domainnameshop

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestFixTTL(t *testing.T) {
	for i, test := range []struct {
		given, expected uint32
	}{
		{1, minAllowedTTL},
		{multiplierTTL*5 - 1, multiplierTTL * 4},
		{maxAllowedTTL + 1, maxAllowedTTL},
		{0, 60},
		{59, 60},
		{60, 60},
		{61, 60},
		{119, 60},
		{120, 120},
		{121, 120},
	} {
		found := fixTTL(test.given)
		if found != test.expected {
			t.Errorf("Test %d: Expected %d, but was %d", i, test.expected, found)
		}
	}
}

func TestToRecordConfig(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		record     domainNameShopRecord
		wantTarget string
	}{
		{"MX", domainNameShopRecord{Host: "@", Type: "MX", Data: "mail.example.net.", TTL: 300, ActualPriority: 10}, "10 mail.example.net."},
		{"SRV", domainNameShopRecord{Host: "_sip._tcp", Type: "SRV", Data: "sip.example.net.", TTL: 300, ActualPriority: 1, ActualWeight: 2, ActualPort: 5060}, "1 2 5060 sip.example.net."},
		{"CAA", domainNameShopRecord{Host: "@", Type: "CAA", Data: "letsencrypt.org", TTL: 300, CAATag: "0", CAAFlag: 1}, `1 issue "letsencrypt.org"`},
		{"TXT", domainNameShopRecord{Host: "@", Type: "TXT", Data: "raw text", TTL: 300}, `"raw text"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.record
			rc, err := toRecordConfig(dc, &record)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
			if rc.Original != &record {
				t.Error("original provider record was not retained")
			}
		})
	}
}
