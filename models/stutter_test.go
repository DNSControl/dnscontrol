package models

import (
	"strings"
	"testing"
)

func Test_doesStutter(t *testing.T) {
	tests := []struct {
		name  string
		rName string
		want  bool
	}{
		{
			name:  "exact domain match should stutter",
			rName: "example.com",
			want:  true,
		},
		{
			name:  "subdomain with dot prefix should stutter",
			rName: "www.example.com",
			want:  true,
		},
		{
			name:  "nested subdomain should stutter",
			rName: "api.staging.example.com",
			want:  true,
		},
		{
			name:  "@ symbol should NOT stutter",
			rName: "@",
			want:  false,
		},
		{
			name:  "different domain should NOT stutter",
			rName: "example.org",
			want:  false,
		},
		{
			name:  "simple subdomain should NOT stutter",
			rName: "www",
			want:  false,
		},
		{
			name:  "similar domain should NOT stutter",
			rName: "testexample.com",
			want:  false,
		},
		{
			name:  "empty name should NOT stutter",
			rName: "",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doesStutter(tt.rName, "example.com"); got != tt.want {
				t.Errorf("stutters(%q, %q) = %v, want %v", tt.rName, "example.com", got, tt.want)
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
