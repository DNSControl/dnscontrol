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

func TestImportRawRecordsStutter(t *testing.T) {
	skip := []map[string]any{{"skip_fqdn_check": "true"}}
	tests := []struct {
		name    string
		zone    string
		label   string
		metas   []map[string]any
		isError bool
	}{
		{"short", "foo.tld", "test", nil, false},
		{"repeats zone", "foo.tld", "test.foo.tld", nil, true},
		{"is zone", "foo.tld", "foo.tld", nil, true},
		{"repeats zone, skipped", "foo.tld", "test.foo.tld", skip, false},
		{"fqdn", "foo.tld", "test.foo.tld.", nil, false},
		{"reverse short", "2.0.192.in-addr.arpa", "1", nil, false},
		{"reverse full name", "2.0.192.in-addr.arpa", "1.2.0.192.in-addr.arpa", nil, false},
		{"reverse is zone", "2.0.192.in-addr.arpa", "2.0.192.in-addr.arpa", nil, true},
		{"reverse repeats zone twice", "2.0.192.in-addr.arpa", "1.2.0.192.in-addr.arpa.2.0.192.in-addr.arpa", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := MustNewDomainConfig(tt.zone)
			dc.RawRecords = []RawRecordConfig{{Type: "TXT", Args: []any{tt.label, "x"}, Metas: tt.metas}}
			config := &DNSConfig{Domains: []*DomainConfig{dc}}
			err := config.ImportRawRecords()
			if (err != nil) != tt.isError {
				t.Errorf("got err=%v, wantErr=%v", err, tt.isError)
			}
			if err != nil && !strings.Contains(err.Error(), "repeats the domain") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
