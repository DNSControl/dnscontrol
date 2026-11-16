package normalize

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestCheckFQDNs(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	dc.AddRecordConfig(dc.MustNewRecordConfig("www", 0, dnsv2.TypeA, "192.0.2.1"))
	dc.AddRecordConfig(dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "192.0.2.2"))
	if errs := checkFQDNs(dc); len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}

	dc.Records[0].NameFQDN = "www.example.net"
	dc.Records[1].NameFQDN = ""
	if errs := checkFQDNs(dc); len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}
