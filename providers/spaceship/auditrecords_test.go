package spaceship

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestAuditRecords_ValidMXAndSRV(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	records := models.Records{
		dc.MustNewRecordConfig("@", 0, "MX", 10, "mail.example.com."),
		dc.MustNewRecordConfig("_sip._tcp", 0, "SRV", 5, 6, 7, "sip.example.com."),
	}
	if errs := AuditRecords(records); len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestAuditRecords_MXNull(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("@", 0, "MX", 0, ".")
	if errs := AuditRecords(models.Records{rc}); len(errs) == 0 {
		t.Error("expected error for null MX (priority=0, target=.), got none")
	}
}

func TestAuditRecords_SRVNullTarget(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("_sip._tcp", 0, "SRV", 15, 65, 75, ".")
	if errs := AuditRecords(models.Records{rc}); len(errs) == 0 {
		t.Error("expected error for SRV with null target, got none")
	}
}
