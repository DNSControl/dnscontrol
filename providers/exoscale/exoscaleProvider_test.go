package exoscale

import (
	"testing"

	egoscale "github.com/exoscale/egoscale/v3"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func makeNativeRecord(rtype, name, content string, priority int64, ttl int64) *egoscale.DNSDomainRecord {
	return &egoscale.DNSDomainRecord{
		Type:     egoscale.DNSDomainRecordType(rtype),
		Name:     name,
		Content:  content,
		Priority: priority,
		Ttl:      ttl,
	}
}

// TestNativeToRecord_ContentPopulated regression: rcontent was declared as "" and never
// assigned from record.Content, so all records got empty content.
func TestNativeToRecord_ContentPopulated(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name       string
		record     *egoscale.DNSDomainRecord
		wantTarget string
		wantType   string
	}{
		{
			name:       "A",
			record:     makeNativeRecord("A", "foo", "1.2.3.4", 0, 300),
			wantTarget: "1.2.3.4",
			wantType:   "A",
		},
		{
			name:       "AAAA",
			record:     makeNativeRecord("AAAA", "foo", "2001:db8::1", 0, 300),
			wantTarget: "2001:db8::1",
			wantType:   "AAAA",
		},
		{
			name:       "TXT",
			record:     makeNativeRecord("TXT", "foo", "hello world", 0, 300),
			wantTarget: `"hello world"`,
			wantType:   "TXT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := nativeToRecord(tt.record, dc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rc == nil {
				t.Fatal("expected record, got nil")
			}
			if rc.Type != tt.wantType {
				t.Errorf("type: got %q, want %q", rc.Type, tt.wantType)
			}
			if rc.GetRDATA().String() != tt.wantTarget {
				t.Errorf("target: got %q, want %q", rc.GetTargetField(), tt.wantTarget)
			}
		})
	}
}

// TestNativeToRecord_SRVNoDoubleDot: API returns content as "weight port target"; we appended
// a trailing dot unconditionally, causing double-dot when the API already returned one.
func TestNativeToRecord_SRVNoDoubleDot(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name    string
		content string // as returned by the API: "weight port target"
	}{
		{"no trailing dot", "20 5060 foo.example.com"},
		{"trailing dot already present", "20 5060 foo.example.com."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := makeNativeRecord("SRV", "_sip._tcp", tt.content, 10, 300)
			rc, err := nativeToRecord(record, dc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rc == nil {
				t.Fatal("expected record, got nil")
			}
			// target should end with exactly one dot
			target := rc.GetTargetField()
			if len(target) >= 2 && target[len(target)-1] == '.' && target[len(target)-2] == '.' {
				t.Errorf("double trailing dot in SRV target: %q", target)
			}
		})
	}
}

// TestNativeToRecord_CNAMENoDoubleDot: same trailing-dot guard for CNAME.
func TestNativeToRecord_CNAMENoDoubleDot(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	for _, content := range []string{"target.example.com", "target.example.com."} {
		record := makeNativeRecord("CNAME", "foo", content, 0, 300)
		rc, err := nativeToRecord(record, dc)
		if err != nil {
			t.Fatalf("unexpected error for content %q: %v", content, err)
		}
		target := rc.GetTargetField()
		if len(target) >= 2 && target[len(target)-1] == '.' && target[len(target)-2] == '.' {
			t.Errorf("double trailing dot in CNAME target for input %q: got %q", content, target)
		}
	}
}

// TestNativeToRecord_SkippedTypes: SOA, NS, and TXT ALIAS mirrors should return nil.
func TestNativeToRecord_SkippedTypes(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	skipped := []*egoscale.DNSDomainRecord{
		makeNativeRecord("SOA", "@", "ns1.example.com", 0, 3600),
		makeNativeRecord("NS", "@", "ns1.example.com.", 0, 3600),
		makeNativeRecord("TXT", "foo", "ALIAS for bar.example.com", 0, 300),
	}

	for _, r := range skipped {
		rc, err := nativeToRecord(r, dc)
		if err != nil {
			t.Errorf("type %s: unexpected error: %v", r.Type, err)
		}
		if rc != nil {
			t.Errorf("type %s: expected nil, got record", r.Type)
		}
	}
}

// TestAuditRecords_PTRRejected: API does not support PTR; provider must reject it.
func TestAuditRecords_PTRRejected(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("4", 300, "PTR", "foo.example.com.")

	errs := AuditRecords(models.Records{rc})
	if len(errs) == 0 {
		t.Error("expected PTR to be rejected, got no errors")
	}
}

// TestAuditRecords_EmptyTXTRejected: API rejects empty TXT values.
func TestAuditRecords_EmptyTXTRejected(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("foo", 300, "TXT", "")

	errs := AuditRecords(models.Records{rc})
	if len(errs) == 0 {
		t.Error("expected empty TXT to be rejected, got no errors")
	}
}

// TestAuditRecords_ValidRecordsPass: common record types should pass without errors.
func TestAuditRecords_ValidRecordsPass(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	records := models.Records{
		dc.MustNewRecordConfig("foo", 300, "A", "1.2.3.4"),
		dc.MustNewRecordConfig("foo", 300, "TXT", "v=spf1 -all"),
	}

	errs := AuditRecords(records)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}
