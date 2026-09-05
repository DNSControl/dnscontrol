package nexdns

import (
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestAuditRecords(t *testing.T) {
	tests := []struct {
		name      string
		record    *models.RecordConfig
		wantCount int
	}{
		{
			name:      "a TXT record with content is fine",
			record:    auditTXT(t, "v=spf1 -all"),
			wantCount: 0,
		},
		{
			name:      "an empty TXT record is rejected",
			record:    auditTXT(t, ""),
			wantCount: 1,
		},
		{
			name:      "a CAA record with a known tag is fine",
			record:    auditCAA(t, 0, "issue", "letsencrypt.org"),
			wantCount: 0,
		},
		{
			name:      "a CAA record with an unknown tag is rejected",
			record:    auditCAA(t, 0, "contactemail", "admin@example.com"),
			wantCount: 1,
		},
		{
			name:      "a CAA value with whitespace is rejected",
			record:    auditCAA(t, 0, "issue", "letsencrypt.org; accounturi=x"),
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := AuditRecords(models.Records{tt.record})
			if len(errs) != tt.wantCount {
				t.Errorf("AuditRecords() returned %d errors, want %d: %v", len(errs), tt.wantCount, errs)
			}
		})
	}
}

func auditTXT(t *testing.T, target string) *models.RecordConfig {
	t.Helper()

	dc := models.MustNewDomainConfig(testOrigin)
	rc := dc.MustNewRecordConfig("@", 0, dnsv2.TypeTXT, "my text")
	if err := rc.SetTargetTXT(target); err != nil {
		t.Fatalf("SetTargetTXT() error = %v", err)
	}
	return rc
}

// The accepted and refused addresses below were read off the API on
// 2026-07-31, not derived from a list of well-known ranges - the two disagree
// on carrier-grade NAT, the documentation ranges and multicast, all of which
// the platform stores happily.
func TestRejectUnroutableIP(t *testing.T) {
	tests := []struct {
		address string
		what    string
		reject  bool
	}{
		{"1.1.1.1", "a public address", false},
		{"172.32.0.1", "just outside RFC 1918", false},
		{"100.64.0.1", "carrier-grade NAT", false},
		{"192.0.2.1", "TEST-NET-1", false},
		{"224.0.0.1", "multicast", false},
		{"2606:4700::1111", "a public v6 address", false},
		{"2001:db8::1", "the v6 documentation range", false},

		{"10.0.0.1", "RFC 1918 /8", true},
		{"172.16.0.1", "RFC 1918 /12", true},
		{"192.168.0.1", "RFC 1918 /16", true},
		{"127.0.0.1", "loopback", true},
		{"169.254.0.1", "link-local", true},
		{"0.0.0.1", "this-network", true},
		{"240.0.0.1", "the reserved /4", true},
		{"255.255.255.255", "broadcast", true},
		{"fd00::1", "a unique-local v6 address", true},
		{"fe80::1", "v6 link-local", true},
		{"::1", "v6 loopback", true},
	}

	for _, tt := range tests {
		t.Run(tt.what, func(t *testing.T) {
			rtype := "A"
			if strings.Contains(tt.address, ":") {
				rtype = "AAAA"
			}

			dc := models.MustNewDomainConfig(testOrigin)
			rc, err := dc.NewRecordConfig("@", 0, rtype, tt.address)
			if err != nil {
				t.Fatalf("SetTarget() error = %v", err)
			}

			err = rejectUnroutableIP(rc)
			if tt.reject && err == nil {
				t.Errorf("%s (%s) was accepted, want rejected", tt.address, tt.what)
			}
			if !tt.reject && err != nil {
				t.Errorf("%s (%s) was rejected: %v", tt.address, tt.what, err)
			}
		})
	}
}

func auditCAA(t *testing.T, flag uint8, tag, target string) *models.RecordConfig {
	t.Helper()

	dc := models.MustNewDomainConfig(testOrigin)
	rc := dc.MustNewRecordConfig("@", 0, dnsv2.TypeCAA, flag, tag, target)
	return rc
}
