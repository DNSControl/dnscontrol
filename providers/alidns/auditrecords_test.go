package alidns

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
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

	dc := models.MustNewDomainConfig("example.com")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := dc.MustNewRecordConfig(tt.label, 0, dnsv2.TypeA, "1.2.3.4")

			err := labelConstraint(rc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("labelConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditRecordsRejectsNonChineseIDNLabel(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("xn--ndaaa", 0, dnsv2.TypeA, "1.2.3.4")

	errs := AuditRecords(models.Records{rc})
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
			dc := models.MustNewDomainConfig("example.com")
			rc := dc.MustNewRecordConfig("a", 0, dnsv2.TypeCNAME, tt.target)

			err := targetConstraint(rc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("targetConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditRecordsRejectsNonChineseIDNCNAMETarget(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("a", 0, dnsv2.TypeCNAME, "xn--ndaaa.com.")

	errs := AuditRecords(models.Records{rc})
	if len(errs) != 1 {
		t.Fatalf("AuditRecords() returned %d errors, want 1", len(errs))
	}
}
