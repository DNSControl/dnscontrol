package ns1

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

// TestBuildRecordNSTarget guards against a regression where NS (and other
// modernized single-target types) were pushed with an empty/origin target.
// The target now lives in RDATA; the legacy .target field is empty and
// CanonicalizeTargets rewrites it to the origin, so buildRecord must read the
// target from RDATA instead of GetTargetField().
func TestBuildRecordNSTarget(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc, err := dc.NewRecordConfig("@", 3600, "NS", "ns1.foo.com.")
	if err != nil {
		t.Fatal(err)
	}
	recs := models.Records{rc}

	// Mimic the normalization that CorrectZoneRecords applies before push.
	// CanonicalizeTargets corrupts the (empty) legacy .target to the origin.
	models.Downcase(recs)
	models.CanonicalizeTargets(recs, dc.Name)

	rec := buildRecord(recs, dc.Name, "")
	if len(rec.Answers) != 1 {
		t.Fatalf("len(answers) = %d, want 1", len(rec.Answers))
	}
	got := rec.Answers[0].Rdata
	if len(got) != 1 || got[0] != "ns1.foo.com." {
		t.Errorf("NS target Rdata = %v, want [ns1.foo.com.]", got)
	}
}

func TestConvertTXT(t *testing.T) {
	t.Parallel()

	dc := models.MustNewDomainConfig("example.com")
	zr := &dns.ZoneRecord{
		Domain:   "example.com",
		Type:     "TXT",
		TTL:      300,
		ShortAns: []string{"v=spf1 include:_spf.google.com ~all"},
	}

	recs, err := convert(zr, dc)
	if err != nil {
		t.Fatalf("convert() error: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("len(recs) = %d, want 1", len(recs))
	}
	if got := recs[0].GetTargetTXTJoined(); got != "v=spf1 include:_spf.google.com ~all" {
		t.Errorf("TXT = %q, want %q", got, "v=spf1 include:_spf.google.com ~all")
	}
}

func TestConvertNAPTR(t *testing.T) {
	t.Parallel()

	dc := models.MustNewDomainConfig("example.com")
	zr := &dns.ZoneRecord{
		Domain: "example.com",
		Type:   "NAPTR",
		TTL:    300,
		ShortAns: []string{
			`100 10 U E2U+sip "!^.*$!sip:customer-service@example.com!" .`,
		},
	}

	recs, err := convert(zr, dc)
	if err != nil {
		t.Fatalf("convert() error: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("len(recs) = %d, want 1", len(recs))
	}
	want := `100 10 "U" "E2U+sip" "!^.*$!sip:customer-service@example.com!" .`
	if got := recs[0].GetRDATA().String(); got != want {
		t.Errorf("NAPTR = %q, want %q", got, want)
	}
}
