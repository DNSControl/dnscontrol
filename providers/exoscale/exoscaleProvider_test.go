package exoscale

import (
	"testing"

	egoscale "github.com/exoscale/egoscale/v3"

	"github.com/DNSControl/dnscontrol/v4/models"
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
			wantTarget: "hello world",
			wantType:   "TXT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := nativeToRecord(tt.record, "example.com")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rc == nil {
				t.Fatal("expected record, got nil")
			}
			if rc.Type != tt.wantType {
				t.Errorf("type: got %q, want %q", rc.Type, tt.wantType)
			}
			if rc.GetTargetField() != tt.wantTarget {
				t.Errorf("target: got %q, want %q", rc.GetTargetField(), tt.wantTarget)
			}
		})
	}
}

// TestNativeToRecord_SRVNoDoubleDot: API returns content as "weight port target"; we appended
// a trailing dot unconditionally, causing double-dot when the API already returned one.
func TestNativeToRecord_SRVNoDoubleDot(t *testing.T) {
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
			rc, err := nativeToRecord(record, "example.com")
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
	for _, content := range []string{"target.example.com", "target.example.com."} {
		record := makeNativeRecord("CNAME", "foo", content, 0, 300)
		rc, err := nativeToRecord(record, "example.com")
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
	skipped := []*egoscale.DNSDomainRecord{
		makeNativeRecord("SOA", "@", "ns1.example.com", 0, 3600),
		makeNativeRecord("NS", "@", "ns1.example.com.", 0, 3600),
		makeNativeRecord("TXT", "foo", "ALIAS for bar.example.com", 0, 300),
	}

	for _, r := range skipped {
		rc, err := nativeToRecord(r, "example.com")
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
	rc := &models.RecordConfig{Type: "PTR"}
	rc.SetLabel("4", "example.com")
	rc.MustSetTarget("foo.example.com.")

	errs := AuditRecords([]*models.RecordConfig{rc})
	if len(errs) == 0 {
		t.Error("expected PTR to be rejected, got no errors")
	}
}

// TestAuditRecords_EmptyTXTRejected: API rejects empty TXT values.
func TestAuditRecords_EmptyTXTRejected(t *testing.T) {
	rc := &models.RecordConfig{Type: "TXT"}
	rc.SetLabel("foo", "example.com")
	_ = rc.SetTargetTXT("")

	errs := AuditRecords([]*models.RecordConfig{rc})
	if len(errs) == 0 {
		t.Error("expected empty TXT to be rejected, got no errors")
	}
}

// TestAuditRecords_ValidRecordsPass: common record types should pass without errors.
func TestAuditRecords_ValidRecordsPass(t *testing.T) {
	var records []*models.RecordConfig

	a := &models.RecordConfig{Type: "A"}
	a.SetLabel("foo", "example.com")
	a.MustSetTarget("1.2.3.4")
	records = append(records, a)

	txt := &models.RecordConfig{Type: "TXT"}
	txt.SetLabel("foo", "example.com")
	_ = txt.SetTargetTXT("v=spf1 -all")
	records = append(records, txt)

	errs := AuditRecords(records)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}
