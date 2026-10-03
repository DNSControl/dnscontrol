package normalize

import (
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

func TestImportTransform(t *testing.T) {
	const transformDouble = "0.0.0.0~1.1.1.1~~9.0.0.0,10.0.0.0"
	const transformSingle = "0.0.0.0~1.1.1.1~~8.0.0.0"

	dcSrc := models.MustNewDomainConfig("stackexchange.com")
	dcSrc.AddRecordConfig(dcSrc.MustNewRecordConfig("*", 0, dnsv2.TypeA, "0.0.2.2"))
	dcSrc.AddRecordConfig(dcSrc.MustNewRecordConfig("www", 0, dnsv2.TypeA, "0.0.1.1"))

	dcDst := models.MustNewDomainConfig("internal")
	d1 := dcDst.MustNewRecordConfig("*.stackexchange.com", 0, dnsv2.TypeA, "0.0.3.3")
	d1.Metadata = map[string]string{"transform_table": transformSingle}
	dcDst.AddRecordConfig(d1)

	d2 := dcSrc.MustNewRecordConfig("@", 0, privatetypes.TypeIMPORTTRANSFORM, transformDouble, 299, "com.internal", "internal")
	d2.Metadata["transform_table"] = transformDouble
	dcDst.AddRecordConfig(d2)

	cfg := &models.DNSConfig{}
	cfg.Domains = append(cfg.Domains, dcSrc, dcDst)
	err := cfg.PostProcess()
	if err != nil {
		t.Fatal(err)
	}

	if errs := ValidateAndNormalizeConfig(cfg); len(errs) != 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.FailNow()
	}
	d := cfg.FindDomain("internal")
	if len(d.Records) != 3 {
		for _, r := range d.Records {
			t.Error(r)
		}
		t.Fatalf("Expected 3 records in internal, but got %d", len(d.Records))
	}
}

// An IMPORT_TRANSFORM that names a domain that is not in the config must be
// reported as an error. It must not crash.
func TestImportTransformMissingDomain(t *testing.T) {
	const transformSingle = "0.0.0.0~1.1.1.1~~8.0.0.0"

	dc := models.MustNewDomainConfig("internal")
	dc.AddRecordConfig(dc.MustNewRecordConfig("www", 0, dnsv2.TypeA, "0.0.3.3"))
	dc.AddRecordConfig(dc.MustNewRecordConfig("@", 0, privatetypes.TypeIMPORTTRANSFORM, transformSingle, 299, "com.internal", "missing.example"))

	cfg := &models.DNSConfig{}
	cfg.Domains = append(cfg.Domains, dc)
	if err := cfg.PostProcess(); err != nil {
		t.Fatal(err)
	}

	var errs []error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ValidateAndNormalizeConfig panicked: %v", r)
			}
		}()
		errs = ValidateAndNormalizeConfig(cfg)
	}()

	for _, err := range errs {
		if strings.Contains(err.Error(), `IMPORT_TRANSFORM mentions non-existent domain "missing.example"`) {
			return
		}
	}
	t.Errorf("expected an error about the non-existent domain, got %v", errs)
}
