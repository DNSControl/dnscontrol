package normalize

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestCheckRecordSetHasMultipleTTLs_err_1type_2ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different ttl per record
	dc.AddTestRC(t, "zzz", 111, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 222, dnsv2.TypeA, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) == 0 {
		t.Error("Expected error on multiple TTLs under the same label, but got none")
	}
}

func TestCheckRecordSetHasMultipleTTLs_noerr_1type_1ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different ttl per record
	dc.AddTestRC(t, "zzz", 111, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 111, dnsv2.TypeA, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (same type, same TTL), but got %d", len(errs))
	}
}

func TestCheckRecordSetHasMultipleTTLs_noerr_2type_2ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different record types, different TTLs
	dc.AddTestRC(t, "zzz", 333, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 444, dnsv2.TypeNS, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (different types, different TTLs), but got %d: %v", len(errs), errs)
	}
}

func TestCheckRecordSetHasMultipleTTLs_noerr_2type_1ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different record types, different TTLs
	dc.AddTestRC(t, "zzz", 333, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 333, dnsv2.TypeNS, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (different types, same TTLs) but got %d: %v", len(errs), errs)
	}
}

func TestCheckRecordSetHasMultipleTTLs_err_3type_2ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different record types, different TTLs
	dc.AddTestRC(t, "zzz", 555, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 555, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 666, dnsv2.TypeNS, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (different types, no errors), but got %d: %v", len(errs), errs)
	}
}

func TestCheckRecordSetHasMultipleTTLs_err_3type_3ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different record types, different TTLs
	dc.AddTestRC(t, "zzz", 777, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 888, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 999, dnsv2.TypeNS, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) != 1 {
		t.Errorf("Expected 0 errors (different types, 1 error), but got %d: %v", len(errs), errs)
	}
}
