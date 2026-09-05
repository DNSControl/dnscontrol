package linode

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestFixTTL(t *testing.T) {
	for i, test := range []struct {
		given, expected uint32
	}{
		{0, 0},
		{1, 30},
		{30, 30},
		{60, 120},
		{299, 300},
		{300, 300},
		{301, 3600},
		{2419202, 2419200},
		{600, 3600},
		{3600, 3600},
	} {
		found := fixTTL(test.given)
		if found != test.expected {
			t.Errorf("Test %d: Expected %d, but was %d", i, test.expected, found)
		}
	}
}

func TestToRc(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		record     domainRecord
		wantTarget string
	}{
		{"A", domainRecord{Name: "www", Type: "A", Target: "192.0.2.1", TTLSec: 300}, "192.0.2.1"},
		{"MX", domainRecord{Name: "@", Type: "MX", Target: "mail.example.net", Priority: 10, TTLSec: 300}, "10 mail.example.net."},
		{"TXT", domainRecord{Name: "@", Type: "TXT", Target: "raw text", TTLSec: 300}, `"raw text"`},
		{"SRV", domainRecord{Name: "_sip._tcp", Type: "SRV", Target: "sip.example.net", Priority: 1, Weight: 2, Port: 5060, TTLSec: 300}, "1 2 5060 sip.example.net."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.record
			rc, err := toRc(dc, &record)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
		})
	}
}
