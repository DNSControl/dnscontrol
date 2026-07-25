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

func TestTXTRecordRoundTripUsesV3RDATA(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	want := dc.MustNewRecordConfig("txt", 300, dnsv2.TypeTXT, `hello "gandi" \\ world`)

	native := recordsToNative(models.Records{want}, dc.Name)
	if len(native) != 1 || len(native[0].RrsetValues) != 1 {
		t.Fatalf("recordsToNative() returned %#v", native)
	}
	got, err := nativeToRecords(dc, native[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("nativeToRecords() returned %d records, want 1", len(got))
	}
	if got[0].GetRDATA().String() != want.GetRDATA().String() {
		t.Errorf("TXT round trip = %q, want %q", got[0].GetRDATA(), want.GetRDATA())
	}
}
