package openprovider

import (
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

const testOrigin = "example.com"

func TestRecordRoundTrip(t *testing.T) {
	longTXT := strings.Repeat("x", 260)
	tests := []struct {
		name         string
		stored       apiRecord
		wantLabel    string
		wantValue    string
		wantPriority int
		wantTarget   string
	}{
		{
			name:       "A at apex",
			stored:     apiRecord{Name: "", Type: "A", Value: "192.0.2.10", TTL: 600},
			wantLabel:  "@",
			wantValue:  "192.0.2.10",
			wantTarget: "192.0.2.10",
		},
		{
			name:       "AAAA with FQDN owner",
			stored:     apiRecord{Name: "v6.example.com", Type: "AAAA", Value: "2001:db8::1", TTL: 3600},
			wantLabel:  "v6",
			wantValue:  "2001:db8::1",
			wantTarget: "2001:db8::1",
		},
		{
			name:       "wildcard CNAME",
			stored:     apiRecord{Name: "*", Type: "CNAME", Value: "target.example.net", TTL: 600},
			wantLabel:  "*",
			wantValue:  "target.example.net",
			wantTarget: "target.example.net.",
		},
		{
			name:         "MX priority",
			stored:       apiRecord{Name: "example.com", Type: "MX", Value: "mail.example.net", Prio: 10, TTL: 600},
			wantLabel:    "@",
			wantValue:    "mail.example.net",
			wantPriority: 10,
			wantTarget:   "mail.example.net.",
		},
		{
			name:         "SRV fields",
			stored:       apiRecord{Name: "_sip._tcp", Type: "SRV", Value: "20 5060 sip.example.net", Prio: 10, TTL: 600},
			wantLabel:    "_sip._tcp",
			wantValue:    "20 5060 sip.example.net",
			wantPriority: 10,
			wantTarget:   "sip.example.net.",
		},
		{
			name:       "CAA quoted value",
			stored:     apiRecord{Name: "", Type: "CAA", Value: `0 issue "letsencrypt.org"`, TTL: 600},
			wantLabel:  "@",
			wantValue:  `0 issue "letsencrypt.org"`,
			wantTarget: "letsencrypt.org",
		},
		{
			name:       "TXT escapes",
			stored:     apiRecord{Name: "txt", Type: "TXT", Value: `"inside \"quote\" and \\ slash"`, TTL: 600},
			wantLabel:  "txt",
			wantValue:  `"inside \"quote\" and \\ slash"`,
			wantTarget: `inside "quote" and \ slash`,
		},
		{
			name:       "long TXT chunks",
			stored:     apiRecord{Name: "long", Type: "TXT", Value: `"` + strings.Repeat("x", 255) + `" "xxxxx"`, TTL: 600},
			wantLabel:  "long",
			wantValue:  `"` + strings.Repeat("x", 255) + `" "xxxxx"`,
			wantTarget: longTXT,
		},
		{
			name:       "SPF presentation",
			stored:     apiRecord{Name: "", Type: "SPF", Value: `"v=spf1 -all"`, TTL: 600},
			wantLabel:  "@",
			wantValue:  `"v=spf1 -all"`,
			wantTarget: "v=spf1 -all",
		},
		{
			name:       "TLSA fields",
			stored:     apiRecord{Name: "_443._tcp", Type: "TLSA", Value: "3 1 1 abcdef0123456789", TTL: 600},
			wantLabel:  "_443._tcp",
			wantValue:  "3 1 1 abcdef0123456789",
			wantTarget: "abcdef0123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := toRecordConfig(tt.stored, testOrigin)
			if err != nil {
				t.Fatalf("toRecordConfig: %v", err)
			}
			if rc.GetLabel() != tt.wantLabel || recordTarget(rc) != tt.wantTarget || rc.TTL != uint32(tt.stored.TTL) {
				t.Errorf("record label=%q target=%q ttl=%d", rc.GetLabel(), recordTarget(rc), rc.TTL)
			}
			if rc.Original != tt.stored {
				t.Errorf("original record was not preserved: %#v", rc.Original)
			}

			request := fromRecordConfig(rc)
			if request.Name != relativeName(tt.wantLabel) || request.Type != tt.stored.Type || request.TTL != tt.stored.TTL || request.Value != tt.wantValue || request.Prio != tt.wantPriority {
				t.Errorf("request = %#v", request)
			}
		})
	}
}

func recordTarget(rc *models.RecordConfig) string {
	if rc.Type == "SPF" {
		return rc.GetTargetTXTJoined()
	}
	if rc.Type == "TLSA" {
		return strings.ToLower(rc.AsTLSA().Certificate)
	}
	return rc.GetTargetField()
}

func TestRelativeHostnameTarget(t *testing.T) {
	rc, err := toRecordConfig(apiRecord{Name: "alias", Type: "CNAME", Value: "www", TTL: 600}, testOrigin)
	if err != nil {
		t.Fatal(err)
	}
	if rc.GetTargetField() != "www.example.com." {
		t.Errorf("target = %q, want www.example.com.", rc.GetTargetField())
	}
}

func TestInvalidSRVValue(t *testing.T) {
	_, err := toRecordConfig(apiRecord{Name: "_sip._tcp", Type: "SRV", Value: "missing fields", TTL: 600}, testOrigin)
	if err == nil || !strings.Contains(err.Error(), "invalid value") {
		t.Fatalf("error = %v", err)
	}
}

func TestAuditRecords(t *testing.T) {
	valid := makeRecord(t, "A", "www", "192.0.2.1")
	validApexNS := makeRecord(t, "NS", "@", "ns1.openprovider.nl.")
	unsupported := makeRecord(t, "NS", "child", "ns.example.net.")
	unsupportedFlag := makeRecord(t, "CAA", "@", `128 issue "example.com"`)
	if errors := AuditRecords(models.Records{valid}); len(errors) != 0 {
		t.Errorf("valid record errors = %v", errors)
	}
	if errors := AuditRecords(models.Records{validApexNS}); len(errors) != 0 {
		t.Errorf("valid apex NS errors = %v", errors)
	}
	if errors := AuditRecords(models.Records{unsupported}); len(errors) != 1 {
		t.Errorf("unsupported record errors = %v", errors)
	}
	if errors := AuditRecords(models.Records{unsupportedFlag}); len(errors) != 1 {
		t.Errorf("unsupported CAA flag errors = %v", errors)
	}
}

func TestNormalizeTTLs(t *testing.T) {
	low := makeRecord(t, "A", "low", "192.0.2.1")
	low.TTL = 300
	high := makeRecord(t, "A", "high", "192.0.2.2")
	high.TTL = 601
	normalizeTTLs(models.Records{low, high})
	if low.TTL != 900 || high.TTL != 900 {
		t.Errorf("TTLs = %d, %d", low.TTL, high.TTL)
	}
}

func makeRecord(t *testing.T, rtype, label, value string) *models.RecordConfig {
	t.Helper()
	dc := &models.DomainConfig{Name: testOrigin}
	rc, err := dc.NewRecordConfigParse(label, minimumTTL, rtype, value)
	if err != nil {
		t.Fatalf("NewRecordConfigParse: %v", err)
	}
	return rc
}

func relativeName(label string) string {
	if label == "@" {
		return ""
	}
	return label
}
