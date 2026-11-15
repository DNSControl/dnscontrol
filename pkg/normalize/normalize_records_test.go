package normalize

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestNormalizeRecords_TTL(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	dc.AddRecordConfig(dc.MustNewRecordConfig("default", 0, dnsv2.TypeA, "192.0.2.1"))
	dc.AddRecordConfig(dc.MustNewRecordConfig("explicit", 600, dnsv2.TypeA, "192.0.2.2"))

	normalizeRecords(dc)

	if got := dc.Records[0].TTL; got != models.DefaultTTL {
		t.Errorf("TTL 0 became %d, want %d", got, models.DefaultTTL)
	}
	if got := dc.Records[1].TTL; got != 600 {
		t.Errorf("TTL 600 became %d, want 600", got)
	}
}
