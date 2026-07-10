package websupport

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func makeRC(rtype, label, target string) *models.RecordConfig {
	rc := &models.RecordConfig{Type: rtype}
	rc.SetLabel(label, "example.com")
	switch rtype {
	case "TXT":
		_ = rc.SetTargetTXT(target)
	case "MX":
		_ = rc.SetTargetMX(10, target)
	case "SRV":
		_ = rc.SetTargetSRV(0, 0, 443, target)
	default:
		_ = rc.SetTarget(target)
	}
	return rc
}

func TestAuditRecords(t *testing.T) {
	tests := []struct {
		name      string
		records   []*models.RecordConfig
		wantCount int
	}{
		{
			name:      "empty",
			records:   []*models.RecordConfig{},
			wantCount: 0,
		},
		{
			name: "supported types are fine",
			records: []*models.RecordConfig{
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
			records:   []*models.RecordConfig{makeRC("NS", "deleg", "ns1.example.net.")},
			wantCount: 1,
		},
		{
			name:      "empty TXT is rejected",
			records:   []*models.RecordConfig{makeRC("TXT", "@", "")},
			wantCount: 1,
		},
		{
			name:      "SRV with null target is rejected",
			records:   []*models.RecordConfig{makeRC("SRV", "_sip._tcp", ".")},
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
