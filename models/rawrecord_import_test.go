package models

import (
	"strings"
	"testing"
)

func TestImportRawRecordsObsoleteMetadata(t *testing.T) {
	tests := []struct {
		name    string
		metas   []map[string]any
		wantErr string
	}{
		{"no metadata", nil, ""},
		{"other metadata", []map[string]any{{"skip_fqdn_check": "true"}}, ""},
		{"obsolete metadata", []map[string]any{{"ignore_name_disable_safety_check": "true"}}, "IGNORE_NAME_DISABLE_SAFETY_CHECK no longer supported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := MustNewDomainConfig("example.com")
			dc.RawRecords = []RawRecordConfig{{Type: "A", Args: []any{"www", "192.0.2.1"}, Metas: tt.metas}}
			config := &DNSConfig{Domains: []*DomainConfig{dc}}
			err := config.ImportRawRecords()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("got err=%v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
