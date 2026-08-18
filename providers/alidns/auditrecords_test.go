package alidns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestLabelConstraint(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		wantErr bool
	}{
		{
			name:  "ascii label",
			label: "www",
		},
		{
			name:  "chinese idn label (punycode)",
			label: "xn--55qx5d", // 公司
		},
		{
			name:    "non-chinese idn label (punycode)",
			label:   "xn--ndaaa", // ööö
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &models.RecordConfig{Type: "A"}
			rc.SetLabel(tt.label, "example.com")

			err := labelConstraint(rc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("labelConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditRecordsRejectsNonChineseIDNLabel(t *testing.T) {
	rc := &models.RecordConfig{Type: "A"}
	rc.SetLabel("xn--ndaaa", "example.com")
	rc.MustSetTarget("1.2.3.4")

	errs := AuditRecords([]*models.RecordConfig{rc})
	if len(errs) != 1 {
		t.Fatalf("AuditRecords() returned %d errors, want 1", len(errs))
	}
}

func TestTargetConstraint(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{
			name:   "ascii target",
			target: "www.example.com.",
		},
		{
			name:   "chinese target",
			target: "xn--55qx5d.",
		},
		{
			name:    "non-chinese idn target",
			target:  "xn--ndaaa.com.",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &models.RecordConfig{Type: "CNAME"}
			rc.SetLabel("a", "example.com")
			rc.MustSetTarget(tt.target)

			err := targetConstraint(rc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("targetConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditRecordsRejectsNonChineseIDNCNAMETarget(t *testing.T) {
	rc := &models.RecordConfig{Type: "CNAME"}
	rc.SetLabel("a", "example.com")
	rc.MustSetTarget("xn--ndaaa.com.")

	errs := AuditRecords([]*models.RecordConfig{rc})
	if len(errs) != 1 {
		t.Fatalf("AuditRecords() returned %d errors, want 1", len(errs))
	}
}
