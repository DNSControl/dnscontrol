package normalize

import (
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
