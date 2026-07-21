package gandiv5

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestRecordsToNative_1(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	dc.AddTestRC(t, dc.LabelFromShort("foo"), 0, dnsv2.TypeA, "1.2.3.4")

	ns := recordsToNative(dc.Records, "example.com")

	if len(ns) != 1 {
		t.Errorf("len(ns) != 1; got=%v", len(ns))
	}
	if len(ns[0].RrsetValues) != 1 {
		t.Errorf("len(ns[0].RrsetValues) != 1; got=%v", ns[0].RrsetValues)
	}
}

func TestRecordsToNative_2(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	dc.AddTestRC(t, dc.LabelFromShort("foo"), 0, dnsv2.TypeA, "1.2.3.4")
	dc.AddTestRC(t, dc.LabelFromShort("foo"), 0, dnsv2.TypeA, "5.6.7.8")

	ns := recordsToNative(dc.Records, "example.com")

	if len(ns) != 1 {
		t.Errorf("len(ns) != 1; got=%v", len(ns))
	}
	if len(ns[0].RrsetValues) != 2 {
		t.Errorf("len(ns[0].RrsetValues) != 2; got=%v", ns[0].RrsetValues)
	}
}
