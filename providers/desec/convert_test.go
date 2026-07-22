package desec

import (
	"reflect"
	"testing"

	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestNativeToRecords(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}

	native := resourceRecord{
		Subname: "www",
		Records: []string{"192.0.2.1", "192.0.2.2"},
		TTL:     300,
		Type:    "A",
	}
	records := nativeToRecords(native, dc)
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	for i, want := range native.Records {
		if got := records[i].GetRDATA().String(); got != want {
			t.Errorf("record %d target = %q, want %q", i, got, want)
		}
		if got := records[i].GetLabel(); got != "www" {
			t.Errorf("record %d label = %q, want %q", i, got, "www")
		}
		if !reflect.DeepEqual(records[i].Original, native) {
			t.Errorf("record %d did not retain its original value", i)
		}
	}
}

func TestNativeToRecordsPreservesRawTXT(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}

	records := nativeToRecords(resourceRecord{
		Subname: "@",
		Records: []string{`"quoted" text`},
		TTL:     300,
		Type:    "TXT",
	}, dc)
	if got, want := models.TXTJoined(records[0].GetRDATA().(dnsrdatav2.TXT)), `"quoted" text`; got != want {
		t.Errorf("TXT target = %q, want %q", got, want)
	}
}

func TestRecordsToNativeUsesShortLabel(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	record := dc.MustNewRecordConfigParse("www", 300, "A", "192.0.2.1")

	native := recordsToNative(models.Records{record})
	if len(native) != 1 {
		t.Fatalf("got %d records, want 1", len(native))
	}
	if got, want := native[0].Subname, "www"; got != want {
		t.Errorf("subname = %q, want %q", got, want)
	}
}
