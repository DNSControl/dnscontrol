package gidinet

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestAuditRecords(t *testing.T) {
	tests := []struct {
		name      string
		records   []*models.RecordConfig
		wantCount int
	}{
		{
			name:      "empty records",
			records:   []*models.RecordConfig{},
			wantCount: 0,
		},
		{
			name: "valid A record",
			records: []*models.RecordConfig{
				makeRC("A", "test", "1.2.3.4"),
			},
			wantCount: 0,
		},
		{
			name: "valid TXT record",
			records: []*models.RecordConfig{
				makeRC("TXT", "test", "hello world"),
			},
			wantCount: 0,
		},
		{
			name: "TXT with double quotes should fail",
			records: []*models.RecordConfig{
				makeRC("TXT", "test", `hello "world"`),
			},
			wantCount: 1,
		},
		{
			name: "empty TXT should fail",
			records: []*models.RecordConfig{
				makeRC("TXT", "test", ""),
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := AuditRecords(tt.records)
			if len(errs) != tt.wantCount {
				t.Errorf("AuditRecords() returned %d errors, want %d: %v", len(errs), tt.wantCount, errs)
			}
		})
	}
}

func makeRC(rtype, label, target string) *models.RecordConfig {
	rc := &models.RecordConfig{
		Type: rtype,
	}
	rc.SetLabel(label, "example.com")
	switch rtype {
	case "TXT":
		rc.SetTargetTXT(target)
	default:
		rc.SetTarget(target)
	}
	return rc
}
