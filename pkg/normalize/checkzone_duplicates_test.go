package normalize

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestCheckDuplicates(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// The only difference is the target:
	dc.AddTestRC(t, "www", 0, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "www", 0, dnsv2.TypeA, "5.5.5.5")
	// The only difference is the rType:
	dc.AddTestRC(t, "aaa", 0, dnsv2.TypeNS, "uniquestring.com.")
	dc.AddTestRC(t, "aaa", 0, dnsv2.TypePTR, "uniquestring.com.")
	// Three records each with a different target.
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns1.foo.com.")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns2.foo.com.")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns3.foo.com.")

	// NOTE: The comparison ignores ttl. Therefore we don't test that.
	errs := checkDuplicates(dc.Records, nil)
	if len(errs) != 0 {
		t.Errorf("Expected duplicate NOT found but found %q", errs)
	}
}

func TestCheckDuplicates_dup_a(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// A records that are exact dupliates.
	dc.AddTestRC(t, "@", 0, dnsv2.TypeA, "1.1.1.1")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeA, "1.1.1.1")

	errs := checkDuplicates(dc.Records, nil)
	if len(errs) == 0 {
		t.Error("Expect duplicate found but found none")
	}
}

func TestCheckDuplicates_dup_ns(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// Three records, the last 2 are duplicates.
	// NB: This is a common issue.
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns1.foo.com.")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns2.foo.com.")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns2.foo.com.")

	errs := checkDuplicates(dc.Records, nil)
	if len(errs) == 0 {
		t.Error("Expect duplicate found but found none")
	}
}
