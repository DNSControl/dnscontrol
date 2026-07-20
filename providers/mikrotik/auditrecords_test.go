package mikrotik

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestAuditRecords_Valid(t *testing.T) {
	records := []*models.RecordConfig{
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
	rc := &models.RecordConfig{}
	rc.SetLabel("@", "example.com")
	rc.Type = "MX"
	_ = rc.SetTargetMX(10, "mail.example.com.")

	errs := AuditRecords([]*models.RecordConfig{rc})
	if len(errs) != 0 {
		t.Errorf("expected 0 errors for valid MX, got %d: %v", len(errs), errs)
	}
}

func TestAuditRecords_MXNull(t *testing.T) {
	rc := &models.RecordConfig{}
	rc.SetLabel("@", "example.com")
	rc.Type = "MX"
	_ = rc.SetTargetMX(0, ".")

	errs := AuditRecords([]*models.RecordConfig{rc})
	if len(errs) == 0 {
		t.Error("expected error for null MX (priority=0, target=.), got none")
	}
}
