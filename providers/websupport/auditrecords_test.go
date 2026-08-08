package websupport

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func makeRC(rtype, label, target string) *models.RecordConfig {
	dc := models.MustNewDomainConfig("example.com")
	switch rtype {
	// case "TXT":
	// 	return dc.MustNewRecordConfig(label, 0, rtype, target)
	case "MX":
		return dc.MustNewRecordConfig(label, 0, rtype, 10, target)
	case "SRV":
		return dc.MustNewRecordConfig(label, 0, rtype, 0, 0, 443, target)
	default:
		return dc.MustNewRecordConfig(label, 0, rtype, target)
	}
}

func TestAuditRecords(t *testing.T) {
	tests := []struct {
		name      string
		records   models.Records
		wantCount int
	}{
		{
			name:      "empty",
			records:   models.Records{},
			wantCount: 0,
		},
		{
			name: "supported types are fine",
			records: models.Records{
				makeRC("A", "@", "1.2.3.4"),
				makeRC("AAAA", "@", "::1"),
				makeRC("CNAME", "www", "example.net."),
				makeRC("MX", "@", "mail.example.com."),
				makeRC("TXT", "@", "hello"),
				makeRC("SRV", "_sip._tcp", "sip.example.com."),
			},
			wantCount: 0,
		},
		{
			name:      "NS is rejected (API silently drops it)",
			records:   models.Records{makeRC("NS", "deleg", "ns1.example.net.")},
			wantCount: 1,
		},
		{
			name:      "empty TXT is rejected",
			records:   models.Records{makeRC("TXT", "@", "")},
			wantCount: 1,
		},
		{
			name:      "SRV with null target is rejected",
			records:   models.Records{makeRC("SRV", "_sip._tcp", ".")},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := AuditRecords(tt.records)
			if len(errs) != tt.wantCount {
				t.Errorf("AuditRecords() = %d errors, want %d: %v", len(errs), tt.wantCount, errs)
			}
		})
	}
}
