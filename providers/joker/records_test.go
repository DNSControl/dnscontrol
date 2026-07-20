package joker

import (
	"reflect"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestParseZoneLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple A record",
			input:    "www A 0 1.2.3.4 300",
			expected: []string{"www", "A", "0", "1.2.3.4", "300"},
		},
		{
			name:     "root domain A record",
			input:    "@ A 0 127.0.0.1 300",
			expected: []string{"@", "A", "0", "127.0.0.1", "300"},
		},
		{
			name:     "CNAME record",
			input:    "mail CNAME 0 example.com. 300",
			expected: []string{"mail", "CNAME", "0", "example.com.", "300"},
		},
		{
			name:     "NS record for subdomain",
			input:    "sub NS 0 ns1.example.com. 300",
			expected: []string{"sub", "NS", "0", "ns1.example.com.", "300"},
		},
		{
			name:     "NS record multiple nameservers",
			input:    "delegate NS 0 ns.provider.com. 3600",
			expected: []string{"delegate", "NS", "0", "ns.provider.com.", "3600"},
		},
		{
			name:     "MX record with priority",
			input:    "@ MX 10 mail.example.com. 300",
			expected: []string{"@", "MX", "10", "mail.example.com.", "300"},
		},
		{
			name:     "TXT record with quoted content",
			input:    "@ TXT 0 \"v=spf1 include:_spf.google.com ~all\" 300",
			expected: []string{"@", "TXT", "0", "\"v=spf1 include:_spf.google.com ~all\"", "300"},
		},
		{
			name:     "TXT record with spaces in quotes",
			input:    "test TXT 0 \"this has multiple spaces\" 300",
			expected: []string{"test", "TXT", "0", "\"this has multiple spaces\"", "300"},
		},
		{
			name:     "TXT record with escaped quotes",
			input:    "test TXT 0 \"value with \\\"escaped\\\" quotes\" 300",
			expected: []string{"test", "TXT", "0", "\"value with \\\"escaped\\\" quotes\"", "300"},
		},
		{
			name:     "SRV record",
			input:    "_sip._tcp SRV 10/20 sip.example.com:5060 300",
			expected: []string{"_sip._tcp", "SRV", "10/20", "sip.example.com:5060", "300"},
		},
		{
			name:     "CAA record",
			input:    "@ CAA 0 0 issue \"letsencrypt.org\" 303",
			expected: []string{"@", "CAA", "0", "0", "issue", "\"letsencrypt.org\"", "303"},
		},
		{
			name:     "NAPTR record",
			input:    "@ NAPTR 100/50 target.example.com. \"u\" \"sip+E2U\" \"!^.*$!sip:info@example.com!\" 313",
			expected: []string{"@", "NAPTR", "100/50", "target.example.com.", "\"u\"", "\"sip+E2U\"", "\"!^.*$!sip:info@example.com!\"", "313"},
		},
		{
			name:     "multiple consecutive spaces",
			input:    "www    A    0    1.2.3.4    300",
			expected: []string{"www", "A", "0", "1.2.3.4", "300"},
		},
		{
			name:     "tabs and spaces mixed",
			input:    "www\tA\t0\t1.2.3.4\t300",
			expected: []string{"www", "A", "0", "1.2.3.4", "300"},
		},
		{
			name:     "quoted content with tabs and spaces",
			input:    "test TXT 0 \"content\twith\ttabs and   spaces\" 300",
			expected: []string{"test", "TXT", "0", "\"content\twith\ttabs and   spaces\"", "300"},
		},
		{
			name:     "escaped backslash",
			input:    "test TXT 0 \"path\\\\to\\\\file\" 300",
			expected: []string{"test", "TXT", "0", "\"path\\\\to\\\\file\"", "300"},
		},
		{
			name:     "empty quoted content",
			input:    "test TXT 0 \"\" 300",
			expected: []string{"test", "TXT", "0", "\"\"", "300"},
		},
		{
			name:     "AAAA record",
			input:    "www AAAA 0 2001:db8::1 300",
			expected: []string{"www", "AAAA", "0", "2001:db8::1", "300"},
		},
		{
			name:     "subdomain with multiple dots",
			input:    "deep.sub.domain A 0 1.2.3.4 300",
			expected: []string{"deep.sub.domain", "A", "0", "1.2.3.4", "300"},
		},
		{
			name:     "TXT record with special characters",
			input:    "test TXT 0 \"key=value;tag=test\" 300",
			expected: []string{"test", "TXT", "0", "\"key=value;tag=test\"", "300"},
		},
		{
			name:     "short record without TTL",
			input:    "www A 0 1.2.3.4",
			expected: []string{"www", "A", "0", "1.2.3.4"},
		},
		{
			name:     "leading and trailing spaces",
			input:    "  www A 0 1.2.3.4 300  ",
			expected: []string{"www", "A", "0", "1.2.3.4", "300"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseZoneLine(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseZoneLine() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseZoneLineEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []string
		expectEmpty bool
	}{
		{
			name:        "empty string",
			input:       "",
			expectEmpty: true,
		},
		{
			name:        "only spaces",
			input:       "   ",
			expectEmpty: true,
		},
		{
			name:        "only tabs",
			input:       "\t\t\t",
			expectEmpty: true,
		},
		{
			name:     "single word",
			input:    "word",
			expected: []string{"word"},
		},
		{
			name:     "unmatched quote at start",
			input:    "\"unclosed quote",
			expected: []string{"\"unclosed quote"},
		},
		{
			name:     "unmatched quote at end",
			input:    "unclosed quote\"",
			expected: []string{"unclosed", "quote\""},
		},
		{
			name:     "consecutive quotes",
			input:    "test \"\" \"value\" more",
			expected: []string{"test", "\"\"", "\"value\"", "more"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseZoneLine(tt.input)

			if tt.expectEmpty {
				if len(got) != 0 {
					t.Errorf("parseZoneLine() = %v, expected empty slice", got)
				}
			} else {
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("parseZoneLine() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}
func TestParseZoneRecords(t *testing.T) {
	api := &jokerProvider{}
	domain := "example.com"

	tests := []struct {
		name      string
		zoneData  string
		wantCount int
		validate  func(t *testing.T, records models.Records)
	}{
		{
			name:      "single A record",
			zoneData:  `www A 0 1.2.3.4 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "A" {
					t.Errorf("Type = %v, want A", records[0].Type)
				}
				if records[0].GetLabelFQDN() != "www.example.com" {
					t.Errorf("Label = %v, want www.example.com", records[0].GetLabelFQDN())
				}
				if records[0].GetTargetField() != "1.2.3.4" {
					t.Errorf("Target = %v, want 1.2.3.4", records[0].GetTargetField())
				}
				if records[0].TTL != 300 {
					t.Errorf("TTL = %v, want 300", records[0].TTL)
				}
			},
		},
		{
			name:      "root domain A record with @",
			zoneData:  `@ A 0 127.0.0.1 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].GetLabelFQDN() != "example.com" {
					t.Errorf("Label = %v, want example.com", records[0].GetLabelFQDN())
				}
				if records[0].GetTargetField() != "127.0.0.1" {
					t.Errorf("Target = %v, want 127.0.0.1", records[0].GetTargetField())
				}
			},
		},
		{
			name:      "AAAA record",
			zoneData:  `www AAAA 0 2001:db8::1 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "AAAA" {
					t.Errorf("Type = %v, want AAAA", records[0].Type)
				}
				if records[0].GetTargetField() != "2001:db8::1" {
					t.Errorf("Target = %v, want 2001:db8::1", records[0].GetTargetField())
				}
			},
		},
		{
			name:      "CNAME record with FQDN",
			zoneData:  `mail CNAME 0 example.com. 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "CNAME" {
					t.Errorf("Type = %v, want CNAME", records[0].Type)
				}
				if records[0].GetTargetField() != "example.com." {
					t.Errorf("Target = %v, want example.com.", records[0].GetTargetField())
				}
			},
		},
		{
			name:      "CNAME record without trailing dot",
			zoneData:  `mail CNAME 0 target.example.com 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].GetTargetField() != "target.example.com." {
					t.Errorf("Target = %v, want target.example.com.", records[0].GetTargetField())
				}
			},
		},
		{
			name:      "NS record",
			zoneData:  `sub NS 0 ns1.example.com. 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "NS" {
					t.Errorf("Type = %v, want NS", records[0].Type)
				}
				if records[0].GetTargetField() != "ns1.example.com." {
					t.Errorf("Target = %v, want ns1.example.com.", records[0].GetTargetField())
				}
			},
		},
		{
			name:      "MX record",
			zoneData:  `@ MX 10 mail.example.com. 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "MX" {
					t.Errorf("Type = %v, want MX", records[0].Type)
				}
				if records[0].MxPreference != 10 {
					t.Errorf("MxPreference = %v, want 10", records[0].MxPreference)
				}
				if records[0].GetTargetField() != "mail.example.com." {
					t.Errorf("Target = %v, want mail.example.com.", records[0].GetTargetField())
				}
			},
		},
		{
			name:      "TXT record with quoted content",
			zoneData:  `@ TXT 0 "v=spf1 include:_spf.google.com ~all" 301`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "TXT" {
					t.Errorf("Type = %v, want TXT", records[0].Type)
				}
				want := "v=spf1 include:_spf.google.com ~all"
				if records[0].GetTargetField() != want {
					t.Errorf("Target = %v, want %v", records[0].GetTargetField(), want)
				}
				wantTTL := uint32(301)
				if records[0].TTL != wantTTL {
					t.Errorf("TTL = %v, want %d", records[0].TTL, wantTTL)
				}
			},
		},
		{
			name:      "TXT record with escaped quotes",
			zoneData:  `test TXT 0 "value with \\\"escaped\\\" quotes" 399`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				want := "value with \\\"escaped\\\" quotes"
				if records[0].GetTargetField() != want {
					t.Errorf("Target = %v, want %v", records[0].GetTargetField(), want)
				}
			},
		},
		{
			name:      "TXT record with escaped backslashes",
			zoneData:  `test TXT 0 "path\\\\to\\\\file" 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				want := "path\\\\to\\\\file"
				if records[0].GetTargetField() != want {
					t.Errorf("Target = %v, want %v", records[0].GetTargetField(), want)
				}
			},
		},
		{
			name:      "SRV record",
			zoneData:  `_sip._tcp SRV 10/20 sip.example.com:5060 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "SRV" {
					t.Errorf("Type = %v, want SRV", records[0].Type)
				}
				if records[0].SrvPriority != 10 {
					t.Errorf("SrvPriority = %v, want 10", records[0].SrvPriority)
				}
				if records[0].SrvWeight != 20 {
					t.Errorf("SrvWeight = %v, want 20", records[0].SrvWeight)
				}
				if records[0].SrvPort != 5060 {
					t.Errorf("SrvPort = %v, want 5060", records[0].SrvPort)
				}
				if records[0].GetTargetField() != "sip.example.com." {
					t.Errorf("Target = %v, want sip.example.com.", records[0].GetTargetField())
				}
			},
		},
		{
			name:      "CAA record",
			zoneData:  `@ CAA 0 issue "letsencrypt.org" 303`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "CAA" {
					t.Errorf("Type = %v, want CAA", records[0].Type)
				}
				if records[0].CaaFlag != 0 {
					t.Errorf("CaaFlag = %v, want 0", records[0].CaaFlag)
				}
				if records[0].CaaTag != "issue" {
					t.Errorf("CaaTag = %v, want issue", records[0].CaaTag)
				}
				if records[0].GetTargetField() != "letsencrypt.org" {
					t.Errorf("Target = %v, want letsencrypt.org", records[0].GetTargetField())
				}
				if records[0].TTL != 303 {
					t.Errorf("TTL = %v, want 303", records[0].TTL)
				}
			},
		},
		{
			name:      "NAPTR record",
			zoneData:  `@ NAPTR 100/50 target.example.com. 315 0 0 "u" "sip+E2U" "!^.*$!sip:info@example.com!"`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "NAPTR" {
					t.Errorf("Type = %v, want NAPTR", records[0].Type)
				}
				if records[0].NaptrOrder != 100 {
					t.Errorf("NaptrOrder = %v, want 100", records[0].NaptrOrder)
				}
				if records[0].NaptrPreference != 50 {
					t.Errorf("NaptrPreference = %v, want 50", records[0].NaptrPreference)
				}
				if records[0].NaptrFlags != "u" {
					t.Errorf("NaptrFlags = %v, want u", records[0].NaptrFlags)
				}
				if records[0].NaptrService != "sip+E2U" {
					t.Errorf("NaptrService = %v, want sip+E2U", records[0].NaptrService)
				}
				if records[0].NaptrRegexp != "!^.*$!sip:info@example.com!" {
					t.Errorf("NaptrRegexp = %v, want !^.*$!sip:info@example.com!", records[0].NaptrRegexp)
				}
				if records[0].GetTargetField() != "target.example.com." {
					t.Errorf("Target = %v, want target.example.com.", records[0].GetTargetField())
				}
				if records[0].TTL != 315 {
					t.Errorf("TTL = %v, want 315", records[0].TTL)
				}
			},
		},
		{
			name: "multiple records",
			zoneData: `www A 0 1.2.3.4 300
mail CNAME 0 example.com. 300
@ MX 10 mail.example.com. 300`,
			wantCount: 3,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "A" {
					t.Errorf("First record type = %v, want A", records[0].Type)
				}
				if records[1].Type != "CNAME" {
					t.Errorf("Second record type = %v, want CNAME", records[1].Type)
				}
				if records[2].Type != "MX" {
					t.Errorf("Third record type = %v, want MX", records[2].Type)
				}
			},
		},
		{
			name: "skip comments and directives",
			zoneData: `# This is a comment
$ORIGIN example.com.
www A 0 1.2.3.4 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].Type != "A" {
					t.Errorf("Type = %v, want A", records[0].Type)
				}
			},
		},
		{
			name: "skip empty lines",
			zoneData: `www A 0 1.2.3.4 300

mail CNAME 0 example.com. 300`,
			wantCount: 2,
			validate: func(t *testing.T, records models.Records) {
				if len(records) != 2 {
					t.Errorf("Record count = %v, want 2", len(records))
				}
			},
		},
		{
			name: "skip malformed lines",
			zoneData: `www A 0
valid A 0 1.2.3.4 300`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].GetLabelFQDN() != "valid.example.com" {
					t.Errorf("Label = %v, want valid.example.com", records[0].GetLabelFQDN())
				}
			},
		},
		{
			name:      "default TTL when missing",
			zoneData:  `www A 0 1.2.3.4`,
			wantCount: 1,
			validate: func(t *testing.T, records models.Records) {
				if records[0].TTL != 300 {
					t.Errorf("TTL = %v, want 300", records[0].TTL)
				}
			},
		},
		{
			name: "skip unsupported record types",
			zoneData: `www A 0 1.2.3.4 300
test UNKNOWN 0 value 300
mail CNAME 0 example.com. 300`,
			wantCount: 2,
			validate: func(t *testing.T, records models.Records) {
				if len(records) != 2 {
					t.Errorf("Record count = %v, want 2", len(records))
				}
			},
		},
		{
			name:      "empty zone data",
			zoneData:  "",
			wantCount: 0,
			validate:  func(t *testing.T, records models.Records) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := api.parseZoneRecords(domain, tt.zoneData)
			if err != nil {
				t.Fatalf("parseZoneRecords() error = %v", err)
			}
			if len(records) != tt.wantCount {
				t.Errorf("parseZoneRecords() returned %d records, want %d", len(records), tt.wantCount)
			}
			if tt.validate != nil {
				tt.validate(t, records)
			}
		})
	}
}
