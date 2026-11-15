package normalize

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestCheckR53WeightedGroupConsistency_noerr_consistent(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc1 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "1.2.3.4")
	rc1.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"}
	dc.AddRecordConfig(rc1)

	rc2 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "2.3.4.5")
	rc2.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"}
	dc.AddRecordConfig(rc2)

	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_noerr_different_set_ids(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc1 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "1.2.3.4")
	rc1.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary"}
	dc.AddRecordConfig(rc1)

	rc2 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "5.6.7.8")
	rc2.Metadata = map[string]string{"r53_weight": "30", "r53_set_identifier": "secondary"}
	dc.AddRecordConfig(rc2)

	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_noerr_no_metadata(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeA, "1.2.3.4")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeA, "5.6.7.8")
	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_err_different_weights(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc1 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "1.2.3.4")
	rc1.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary"}
	dc.AddRecordConfig(rc1)

	rc2 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "2.3.4.5")
	rc2.Metadata = map[string]string{"r53_weight": "50", "r53_set_identifier": "primary"}
	dc.AddRecordConfig(rc2)

	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error for inconsistent weights but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_err_different_health_checks(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc1 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "1.2.3.4")
	rc1.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"}
	dc.AddRecordConfig(rc1)

	rc2 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "2.3.4.5")
	rc2.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-2"}
	dc.AddRecordConfig(rc2)

	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error for inconsistent health checks but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_err_both_inconsistent(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc1 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "1.2.3.4")
	rc1.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"}
	dc.AddRecordConfig(rc1)

	rc2 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "2.3.4.5")
	rc2.Metadata = map[string]string{"r53_weight": "50", "r53_set_identifier": "primary", "r53_health_check_id": "hc-2"}
	dc.AddRecordConfig(rc2)

	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 2 {
		t.Errorf("Expected 2 errors (weight + health check) but got %d: %v", len(errs), errs)
	}
}
