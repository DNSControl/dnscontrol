package normalize

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func makeRC(label, domain, target string, rc models.RecordConfig) *models.RecordConfig {
	rc.SetLabel(label, domain)
	rc.MustSetTarget(target)
	return &rc
}

func TestImportTransform(t *testing.T) {
	const transformDouble = "0.0.0.0~1.1.1.1~~9.0.0.0,10.0.0.0"
	const transformSingle = "0.0.0.0~1.1.1.1~~8.0.0.0"

	src := models.MustNewDomainConfig("stackexchange.com")
	s1 := makeRC("*", "stackexchange.com", "0.0.2.2", models.RecordConfig{Type: "A"})
	s2 := makeRC("www", "stackexchange.com", "0.0.1.1", models.RecordConfig{Type: "A"})
	src.Records = []*models.RecordConfig{s1, s2}

	dst := models.MustNewDomainConfig("internal")
	d1 := makeRC("*.stackexchange.com", "*.stackexchange.com.internal", "0.0.3.3", models.RecordConfig{Type: "A"})
	d1.Metadata = map[string]string{"transform_table": transformSingle}
	d2 := makeRC("@", "internal", "stackexchange.com", models.RecordConfig{Type: "IMPORT_TRANSFORM"})
	d2.Metadata = map[string]string{"transform_table": transformDouble}
	dst.Records = []*models.RecordConfig{d1, d2}

	cfg := &models.DNSConfig{
		Domains: []*models.DomainConfig{src, dst},
	}
	err := cfg.PostProcess()
	if err != nil {
		t.Fatal(err)
	}
	// No need to call rtypecontrol.FixLegacyDC here.

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
