package normalize

import (
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

// Records added by the apply phase get the per-record checks. Importing a
// subdomain into its parent produces a label that repeats the zone name, which
// checkLabel rejects.
func TestGeneratedRecordsAreChecked(t *testing.T) {
	const transform = "0.0.0.0~255.255.255.255~~0.0.0.0"

	src := models.MustNewDomainConfig("internal.example.com")
	src.AddRecordConfig(src.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1"))

	dst := models.MustNewDomainConfig("example.com")
	dst.AddRecordConfig(dst.MustNewRecordConfig("@", 300, privatetypes.TypeIMPORTTRANSFORM, transform, 300, "", src.Name))

	cfg := &models.DNSConfig{Domains: []*models.DomainConfig{src, dst}}
	if err := cfg.PostProcess(); err != nil {
		t.Fatal(err)
	}

	errs := ValidateAndNormalizeConfig(cfg)
	if len(dst.Records) != 1 || dst.Records[0].GetLabel() != "www.internal.example.com" {
		t.Fatalf("unexpected records in %s: %v", dst.Name, dst.Records)
	}
	for _, err := range errs {
		if strings.Contains(err.Error(), "www.internal.example.com") {
			return
		}
	}
	t.Errorf("expected an error about the label of the imported record, got %v", errs)
}

// A record type that a provider does not support is reported once.
func TestUnsupportedTypeReportedOnce(t *testing.T) {
	dc := lineDomain("S5_EMPTY")
	dc.AddRecordConfig(dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1"))
	errs := validateDomain(t, dc)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "uses A records") {
		t.Errorf("expected exactly one unsupported-type error, got %v", errs)
	}
}
