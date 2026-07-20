package hedns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func makeRC(rtype, label, target string) *models.RecordConfig {
	rc := &models.RecordConfig{
		Type: rtype,
	}
	rc.SetLabel(label, "example.com")
	switch rtype {
	case "TXT":
		rc.SetTargetTXT(target)
	case "MX":
		rc.SetTargetMX(10, target)
	default:
		rc.SetTarget(target)
	}
	return rc
}

func withMeta(rc *models.RecordConfig, key, value string) *models.RecordConfig {
	if rc.Metadata == nil {
		rc.Metadata = map[string]string{}
	}
	rc.Metadata[key] = value
	return rc
}

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

		// Allowed: A, AAAA, TXT with hedns_dynamic
		{
			name:      "A with dynamic on",
			records:   []*models.RecordConfig{withMeta(makeRC("A", "dyn", "1.2.3.4"), metaDynamic, "on")},
			wantCount: 0,
		},
		{
			name:      "AAAA with dynamic on",
			records:   []*models.RecordConfig{withMeta(makeRC("AAAA", "dyn6", "::1"), metaDynamic, "on")},
			wantCount: 0,
		},
		{
			name:      "TXT with dynamic on",
			records:   []*models.RecordConfig{withMeta(makeRC("TXT", "dyntxt", "val"), metaDynamic, "on")},
			wantCount: 0,
		},
		{
			name:      "A with dynamic off is allowed",
			records:   []*models.RecordConfig{withMeta(makeRC("A", "s", "1.2.3.4"), metaDynamic, "off")},
			wantCount: 0,
		},

		// Allowed: A, AAAA with hedns_ddns_key
		{
			name:      "A with ddns key",
			records:   []*models.RecordConfig{withMeta(makeRC("A", "k", "1.2.3.4"), metaDDNSKey, "tok")},
			wantCount: 0,
		},
		{
			name:      "AAAA with ddns key",
			records:   []*models.RecordConfig{withMeta(makeRC("AAAA", "k6", "::1"), metaDDNSKey, "tok")},
			wantCount: 0,
		},

		// Rejected: unsupported types with hedns_dynamic
		{
			name:      "CNAME with dynamic on rejected",
			records:   []*models.RecordConfig{withMeta(makeRC("CNAME", "c", "example.com."), metaDynamic, "on")},
			wantCount: 1,
		},
		{
			name:      "MX with dynamic on rejected",
			records:   []*models.RecordConfig{withMeta(makeRC("MX", "m", "mail.example.com."), metaDynamic, "on")},
			wantCount: 1,
		},
		{
			name:      "SRV with dynamic on rejected",
			records:   []*models.RecordConfig{withMeta(makeRC("CNAME", "srv", "example.com."), metaDynamic, "on")},
			wantCount: 1,
		},

		// Rejected: unsupported types with hedns_ddns_key
		{
			name:      "CNAME with ddns key rejected",
			records:   []*models.RecordConfig{withMeta(makeRC("CNAME", "c", "example.com."), metaDDNSKey, "tok")},
			wantCount: 1,
		},

		// Non-dynamic metadata on unsupported types is fine
		{
			name:      "CNAME with dynamic off is allowed",
			records:   []*models.RecordConfig{withMeta(makeRC("CNAME", "c", "example.com."), metaDynamic, "off")},
			wantCount: 0,
		},
		{
			name:      "CNAME without metadata is allowed",
			records:   []*models.RecordConfig{makeRC("CNAME", "c", "example.com.")},
			wantCount: 0,
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
