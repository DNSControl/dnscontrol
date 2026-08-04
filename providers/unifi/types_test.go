package unifi

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

const testDomain = "example.com"

var testDC = models.MustNewDomainConfig(testDomain)

func TestLegacyToRecord(t *testing.T) {
	tests := []struct {
		name       string
		in         *legacyDNSRecord
		wantType   string
		wantLabel  string
		wantTTL    uint32
		wantTarget string
	}{
		{
			name:       "A",
			in:         &legacyDNSRecord{Key: "www.example.com", RecordType: "A", Value: "1.2.3.4", TTL: 3600},
			wantType:   "A",
			wantLabel:  "www",
			wantTTL:    3600,
			wantTarget: "1.2.3.4",
		},
		{
			name:       "AAAA",
			in:         &legacyDNSRecord{Key: "www.example.com", RecordType: "AAAA", Value: "2001:db8::1", TTL: 3600},
			wantType:   "AAAA",
			wantLabel:  "www",
			wantTTL:    3600,
			wantTarget: "2001:db8::1",
		},
		{
			name:       "CNAME gets trailing dot",
			in:         &legacyDNSRecord{Key: "alias.example.com", RecordType: "CNAME", Value: "target.example.com", TTL: 300},
			wantType:   "CNAME",
			wantLabel:  "alias",
			wantTTL:    300,
			wantTarget: "target.example.com.",
		},
		{
			name:       "NS gets trailing dot",
			in:         &legacyDNSRecord{Key: "example.com", RecordType: "NS", Value: "ns1.example.com", TTL: 300},
			wantType:   "NS",
			wantLabel:  "@",
			wantTTL:    300,
			wantTarget: "ns1.example.com.",
		},
		{
			name:       "MX keeps priority and dot",
			in:         &legacyDNSRecord{Key: "example.com", RecordType: "MX", Value: "mail.example.com", Priority: 10, TTL: 300},
			wantType:   "MX",
			wantLabel:  "@",
			wantTTL:    300,
			wantTarget: "10 mail.example.com.",
		},
		{
			name:       "TXT",
			in:         &legacyDNSRecord{Key: "example.com", RecordType: "TXT", Value: "v=spf1 -all", TTL: 300},
			wantType:   "TXT",
			wantLabel:  "@",
			wantTTL:    300,
			wantTarget: `"v=spf1 -all"`,
		},
		{
			name:       "SRV",
			in:         &legacyDNSRecord{Key: "_sip._tcp.example.com", RecordType: "SRV", Value: "sip.example.com", Priority: 1, Weight: 5, Port: 5060, TTL: 300},
			wantType:   "SRV",
			wantLabel:  "_sip._tcp",
			wantTTL:    300,
			wantTarget: "1 5 5060 sip.example.com.",
		},
		{
			name:       "TTL 0 defaults to 300",
			in:         &legacyDNSRecord{Key: "www.example.com", RecordType: "A", Value: "1.2.3.4", TTL: 0},
			wantType:   "A",
			wantLabel:  "www",
			wantTTL:    300,
			wantTarget: "1.2.3.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := legacyToRecord(testDC, tt.in)
			if err != nil {
				t.Fatalf("legacyToRecord() unexpected error: %v", err)
			}
			if rc.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", rc.Type, tt.wantType)
			}
			if rc.Name != tt.wantLabel {
				t.Errorf("Name = %q, want %q", rc.Name, tt.wantLabel)
			}
			if rc.TTL != tt.wantTTL {
				t.Errorf("TTL = %d, want %d", rc.TTL, tt.wantTTL)
			}
			if got := rc.GetRDATA().String(); got != tt.wantTarget {
				t.Errorf("target = %q, want %q", got, tt.wantTarget)
			}
			// The raw provider record must be retained for later ID lookups.
			if rc.Original != tt.in {
				t.Errorf("Original not preserved")
			}
		})
	}
}

func TestLegacyToRecordUnsupported(t *testing.T) {
	_, err := legacyToRecord(testDC, &legacyDNSRecord{Key: "example.com", RecordType: "CAA", Value: "0 issue \"ca.example.net\""})
	if err == nil {
		t.Fatal("expected error for unsupported record type, got nil")
	}
}

func TestNewToRecord(t *testing.T) {
	tests := []struct {
		name       string
		in         *dnsPolicyRecord
		wantType   string
		wantLabel  string
		wantTTL    uint32
		wantTarget string
	}{
		{
			name:       "A",
			in:         &dnsPolicyRecord{Type: NewAPITypeA, Domain: "www.example.com", IPv4Address: "1.2.3.4", TTLSeconds: 3600},
			wantType:   "A",
			wantLabel:  "www",
			wantTTL:    3600,
			wantTarget: "1.2.3.4",
		},
		{
			name:       "AAAA",
			in:         &dnsPolicyRecord{Type: NewAPITypeAAAA, Domain: "www.example.com", IPv6Address: "2001:db8::1", TTLSeconds: 3600},
			wantType:   "AAAA",
			wantLabel:  "www",
			wantTTL:    3600,
			wantTarget: "2001:db8::1",
		},
		{
			name:       "CNAME gets trailing dot",
			in:         &dnsPolicyRecord{Type: NewAPITypeCNAME, Domain: "alias.example.com", TargetDomain: "target.example.com", TTLSeconds: 300},
			wantType:   "CNAME",
			wantLabel:  "alias",
			wantTTL:    300,
			wantTarget: "target.example.com.",
		},
		{
			name:       "MX keeps priority and dot",
			in:         &dnsPolicyRecord{Type: NewAPITypeMX, Domain: "example.com", MailServerDomain: "mail.example.com", Priority: 10, TTLSeconds: 300},
			wantType:   "MX",
			wantLabel:  "@",
			wantTTL:    300,
			wantTarget: "10 mail.example.com.",
		},
		{
			name:       "TXT",
			in:         &dnsPolicyRecord{Type: NewAPITypeTXT, Domain: "example.com", Text: "v=spf1 -all", TTLSeconds: 300},
			wantType:   "TXT",
			wantLabel:  "@",
			wantTTL:    300,
			wantTarget: `"v=spf1 -all"`,
		},
		{
			name:       "SRV",
			in:         &dnsPolicyRecord{Type: NewAPITypeSRV, Domain: "example.com", Service: "_sip", Protocol: "_tcp", ServerDomain: "sip.example.com", Priority: 1, Weight: 5, Port: 5060, TTLSeconds: 300},
			wantType:   "SRV",
			wantLabel:  "_sip._tcp",
			wantTTL:    300,
			wantTarget: "1 5 5060 sip.example.com.",
		},
		{
			name:       "TTL 0 defaults to 300",
			in:         &dnsPolicyRecord{Type: NewAPITypeA, Domain: "www.example.com", IPv4Address: "1.2.3.4", TTLSeconds: 0},
			wantType:   "A",
			wantLabel:  "www",
			wantTTL:    300,
			wantTarget: "1.2.3.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := newToRecord(testDC, tt.in)
			if err != nil {
				t.Fatalf("newToRecord() unexpected error: %v", err)
			}
			if rc.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", rc.Type, tt.wantType)
			}
			if rc.Name != tt.wantLabel {
				t.Errorf("Name = %q, want %q", rc.Name, tt.wantLabel)
			}
			if rc.TTL != tt.wantTTL {
				t.Errorf("TTL = %d, want %d", rc.TTL, tt.wantTTL)
			}
			if got := rc.GetRDATA().String(); got != tt.wantTarget {
				t.Errorf("target = %q, want %q", got, tt.wantTarget)
			}
			if rc.Original != tt.in {
				t.Errorf("Original not preserved")
			}
		})
	}
}

func TestNewToRecordUnsupported(t *testing.T) {
	_, err := newToRecord(testDC, &dnsPolicyRecord{Type: "CAA_RECORD", Domain: "example.com"})
	if err == nil {
		t.Fatal("expected error for unsupported new API record type, got nil")
	}
}

// TestRecordToLegacyMap converts a provider record to a RecordConfig and back to
// the legacy API map, verifying the API is given exactly the fields it expects.
func TestRecordToLegacyMap(t *testing.T) {
	tests := []struct {
		name  string
		in    *legacyDNSRecord
		check func(t *testing.T, m map[string]any)
	}{
		{
			name: "A includes ttl",
			in:   &legacyDNSRecord{Key: "www.example.com", RecordType: "A", Value: "1.2.3.4", TTL: 3600},
			check: func(t *testing.T, m map[string]any) {
				wantMapKV(t, m, "record_type", "A")
				wantMapKV(t, m, "key", "www.example.com")
				wantMapKV(t, m, "value", "1.2.3.4")
				wantMapKV(t, m, "ttl", 3600)
			},
		},
		{
			name: "CNAME strips trailing dot",
			in:   &legacyDNSRecord{Key: "alias.example.com", RecordType: "CNAME", Value: "target.example.com", TTL: 300},
			check: func(t *testing.T, m map[string]any) {
				wantMapKV(t, m, "value", "target.example.com")
			},
		},
		{
			name: "MX carries priority and no ttl",
			in:   &legacyDNSRecord{Key: "example.com", RecordType: "MX", Value: "mail.example.com", Priority: 10, TTL: 300},
			check: func(t *testing.T, m map[string]any) {
				wantMapKV(t, m, "value", "mail.example.com")
				wantMapKV(t, m, "priority", 10)
				if _, ok := m["ttl"]; ok {
					t.Errorf("MX map should not carry a ttl field, got %v", m["ttl"])
				}
			},
		},
		{
			name: "TXT uses joined value",
			in:   &legacyDNSRecord{Key: "example.com", RecordType: "TXT", Value: "v=spf1 -all", TTL: 300},
			check: func(t *testing.T, m map[string]any) {
				wantMapKV(t, m, "value", "v=spf1 -all")
			},
		},
		{
			name: "SRV carries priority weight port",
			in:   &legacyDNSRecord{Key: "_sip._tcp.example.com", RecordType: "SRV", Value: "sip.example.com", Priority: 1, Weight: 5, Port: 5060, TTL: 300},
			check: func(t *testing.T, m map[string]any) {
				wantMapKV(t, m, "value", "sip.example.com")
				wantMapKV(t, m, "priority", 1)
				wantMapKV(t, m, "weight", 5)
				wantMapKV(t, m, "port", 5060)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := legacyToRecord(testDC, tt.in)
			if err != nil {
				t.Fatalf("legacyToRecord() error: %v", err)
			}
			m, err := recordToLegacyMap(rc)
			if err != nil {
				t.Fatalf("recordToLegacyMap() error: %v", err)
			}
			if m["enabled"] != true {
				t.Errorf("enabled = %v, want true", m["enabled"])
			}
			tt.check(t, m)
		})
	}
}

func TestRecordToLegacyMapUnsupported(t *testing.T) {
	_, err := recordToLegacyMap(recordWithType("CAA"))
	if err == nil {
		t.Fatal("expected error for unsupported record type, got nil")
	}
}

// TestRecordToNew converts a provider record to a RecordConfig and back to the
// new API record, verifying the type-specific fields are populated.
func TestRecordToNew(t *testing.T) {
	tests := []struct {
		name  string
		in    *dnsPolicyRecord
		check func(t *testing.T, r *dnsPolicyRecord)
	}{
		{
			name: "A",
			in:   &dnsPolicyRecord{Type: NewAPITypeA, Domain: "www.example.com", IPv4Address: "1.2.3.4", TTLSeconds: 3600},
			check: func(t *testing.T, r *dnsPolicyRecord) {
				if r.Type != NewAPITypeA || r.IPv4Address != "1.2.3.4" || r.Domain != "www.example.com" || r.TTLSeconds != 3600 {
					t.Errorf("unexpected A record: %+v", r)
				}
			},
		},
		{
			name: "CNAME strips trailing dot",
			in:   &dnsPolicyRecord{Type: NewAPITypeCNAME, Domain: "alias.example.com", TargetDomain: "target.example.com", TTLSeconds: 300},
			check: func(t *testing.T, r *dnsPolicyRecord) {
				if r.Type != NewAPITypeCNAME || r.TargetDomain != "target.example.com" {
					t.Errorf("unexpected CNAME record: %+v", r)
				}
			},
		},
		{
			name: "MX",
			in:   &dnsPolicyRecord{Type: NewAPITypeMX, Domain: "example.com", MailServerDomain: "mail.example.com", Priority: 10, TTLSeconds: 300},
			check: func(t *testing.T, r *dnsPolicyRecord) {
				if r.Type != NewAPITypeMX || r.MailServerDomain != "mail.example.com" || r.Priority != 10 || r.TTLSeconds != 0 {
					t.Errorf("unexpected MX record: %+v", r)
				}
			},
		},
		{
			name: "SRV",
			in:   &dnsPolicyRecord{Type: NewAPITypeSRV, Domain: "example.com", Service: "_sip", Protocol: "_tcp", ServerDomain: "sip.example.com", Priority: 1, Weight: 5, Port: 5060, TTLSeconds: 300},
			check: func(t *testing.T, r *dnsPolicyRecord) {
				if r.Type != NewAPITypeSRV || r.Service != "_sip" || r.Protocol != "_tcp" || r.Domain != "example.com" || r.ServerDomain != "sip.example.com" || r.Priority != 1 || r.Weight != 5 || r.Port != 5060 || r.TTLSeconds != 0 {
					t.Errorf("unexpected SRV record: %+v", r)
				}
			},
		},
		{
			name: "TTL 0 defaults to 300",
			in:   &dnsPolicyRecord{Type: NewAPITypeA, Domain: "www.example.com", IPv4Address: "1.2.3.4", TTLSeconds: 0},
			check: func(t *testing.T, r *dnsPolicyRecord) {
				if r.TTLSeconds != 300 {
					t.Errorf("TTLSeconds = %d, want 300", r.TTLSeconds)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := newToRecord(testDC, tt.in)
			if err != nil {
				t.Fatalf("newToRecord() error: %v", err)
			}
			r, err := recordToNew(rc)
			if err != nil {
				t.Fatalf("recordToNew() error: %v", err)
			}
			if !r.Enabled {
				t.Errorf("Enabled = false, want true")
			}
			tt.check(t, r)
		})
	}
}

func TestRecordToNewUnsupported(t *testing.T) {
	_, err := recordToNew(recordWithType("CAA"))
	if err == nil {
		t.Fatal("expected error for unsupported record type, got nil")
	}
}

func TestGetRecordID(t *testing.T) {
	legacy := recordWithOriginal(&legacyDNSRecord{ID: "legacy-123"})
	if got := getRecordID(legacy); got != "legacy-123" {
		t.Errorf("getRecordID(legacy) = %q, want %q", got, "legacy-123")
	}

	newRec := recordWithOriginal(&dnsPolicyRecord{ID: "new-456"})
	if got := getRecordID(newRec); got != "new-456" {
		t.Errorf("getRecordID(new) = %q, want %q", got, "new-456")
	}

	if got := getRecordID(new(models.RecordConfig)); got != "" {
		t.Errorf("getRecordID(nil Original) = %q, want empty", got)
	}

	if got := getRecordID(recordWithOriginal("not-a-record")); got != "" {
		t.Errorf("getRecordID(unknown Original) = %q, want empty", got)
	}
}

func recordWithType(rtype string) *models.RecordConfig {
	rc := new(models.RecordConfig)
	rc.Type = rtype
	return rc
}

func recordWithOriginal(original any) *models.RecordConfig {
	rc := new(models.RecordConfig)
	rc.Original = original
	return rc
}

func wantMapKV(t *testing.T, m map[string]any, key string, want any) {
	t.Helper()
	got, ok := m[key]
	if !ok {
		t.Errorf("map missing key %q", key)
		return
	}
	if got != want {
		t.Errorf("map[%q] = %v (%T), want %v (%T)", key, got, got, want, want)
	}
}
