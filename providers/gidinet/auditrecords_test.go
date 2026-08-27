package gidinet

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestAuditRecords(t *testing.T) {
	tests := []struct {
		name      string
		records   models.Records
		wantCount int
	}{
		{
			name:      "empty records",
			records:   models.Records{},
			wantCount: 0,
		},
		{
			name: "valid A record",
			records: models.Records{
				makeRC("A", "test", "1.2.3.4"),
			},
			wantCount: 0,
		},
		{
			name: "valid TXT record",
			records: models.Records{
				makeRC("TXT", "test", "hello world"),
			},
			wantCount: 0,
		},
		{
			name: "TXT with double quotes should fail",
			records: models.Records{
				makeRC("TXT", "test", `hello "world"`),
			},
			wantCount: 1,
		},
		{
			name: "empty TXT should fail",
			records: models.Records{
				makeRC("TXT", "test", ""),
			},
			wantCount: 1,
		},
		{
			name: "valid SRV record",
			records: models.Records{
				makeRC("SRV", "_sip._tcp", "foo.com."),
			},
			wantCount: 0,
		},
		{
			name: "SRV with null target should fail",
			records: models.Records{
				makeRC("SRV", "_sip._tcp", "."),
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
	dc, _ := models.NewDomainConfig("example.com")
	switch rtype {
	case "SRV":
		return dc.MustNewRecordConfig(label, 300, rtype, 0, 0, 0, target)
	// case "TXT":
	// 	return dc.MustNewRecordConfig(label, 300, rtype, target)
	default:
		return dc.MustNewRecordConfig(label, 300, rtype, target)
	}
}
