package mikrotik

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestAuditRecords_Valid(t *testing.T) {
	records := models.Records{
		makeRC("A", "host", "example.com", "10.0.0.1"),
		makeRC("CNAME", "alias", "example.com", "host.example.com."),
		makeRC("TXT", "@", "example.com", "v=spf1 ~all"),
	}

	errs := AuditRecords(records)
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestAuditRecords_MXValid(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("@", 0, "MX", 10, "mail.example.com.")

	errs := AuditRecords(models.Records{rc})
	if len(errs) != 0 {
		t.Errorf("expected 0 errors for valid MX, got %d: %v", len(errs), errs)
	}
}

func TestAuditRecords_MXNull(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("@", 0, "MX", 0, ".")

	errs := AuditRecords(models.Records{rc})
	if len(errs) == 0 {
		t.Error("expected error for null MX (priority=0, target=.), got none")
	}
}
