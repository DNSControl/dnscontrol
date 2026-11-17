package normalize

import (
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

// IMPORT_TRANSFORM labels are generated from the source records' full names,
// so they are not checked for repeating the zone name. Importing a subdomain
// into its parent must not be rejected.
func TestImportTransformSkipsStutterCheck(t *testing.T) {
	const transform = "0.0.0.0~255.255.255.255~~0.0.0.0"

	src := models.MustNewDomainConfig("internal.example.com")
	src.AddRecordConfig(src.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1"))

	dst := models.MustNewDomainConfig("example.com")
	dst.AddRecordConfig(dst.MustNewRecordConfig("@", 300, privatetypes.TypeIMPORTTRANSFORM, transform, 300, "", src.Name))

	cfg := &models.DNSConfig{Domains: []*models.DomainConfig{src, dst}}
	if err := cfg.PostProcess(); err != nil {
		t.Fatal(err)
	}

	for _, err := range ValidateAndNormalizeConfig(cfg) {
		if strings.Contains(err.Error(), "repeats the domain") {
			t.Errorf("IMPORT_TRANSFORM output was stutter-checked: %v", err)
		}
	}
	if !dst.Records.HasRecordTypeName("A", "www.internal.example.com") {
		t.Errorf("imported record missing from %s", dst.Name)
	}
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
