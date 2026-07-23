package ns1

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

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
