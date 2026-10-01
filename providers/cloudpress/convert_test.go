package cloudpress

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestFromRecordConfigStripsTrailingDot(t *testing.T) {
	rc := &models.RecordConfig{Type: "CNAME", TTL: 300}
	rc.SetLabelFromFQDN("www.example.com", "example.com")
	rc.MustSetTarget("target.example.com.")

	r := fromRecordConfig(rc)
	if r.RecordType != recordTypeCNAME {
		t.Fatalf("expected record_type=%d; got=%d", recordTypeCNAME, r.RecordType)
	}
	if r.Value != "target.example.com" {
		t.Fatalf("expected trailing dot stripped; got=%q", r.Value)
	}
	// CloudPress wants the fully-qualified name and strips the zone itself.
	if r.Name != "www.example.com" {
		t.Fatalf("expected name=www.example.com; got=%q", r.Name)
	}
}

func TestFromRecordConfigMX(t *testing.T) {
	rc := &models.RecordConfig{Type: "MX", TTL: 3600}
	rc.SetLabelFromFQDN("example.com", "example.com")
	if err := rc.SetTargetMX(10, "mail.example.com."); err != nil {
		t.Fatal(err)
	}

	r := fromRecordConfig(rc)
	if r.Priority == nil || *r.Priority != 10 {
		t.Fatalf("expected priority=10; got=%v", r.Priority)
	}
	if r.Value != "mail.example.com" {
		t.Fatalf("expected mail.example.com; got=%q", r.Value)
	}
	// The apex is sent as the bare zone name.
	if r.Name != "example.com" {
		t.Fatalf("expected apex name=example.com; got=%q", r.Name)
	}
}

func TestToRecordConfigApex(t *testing.T) {
	// CloudPress returns the bare zone name for apex records.
	rec := &record{RecordType: recordTypeTXT, Name: "example.com", Value: "hello", TTL: 300}

	rc, err := toRecordConfig("example.com", rec)
	if err != nil {
		t.Fatalf("toRecordConfig returned error: %v", err)
	}
	if rc.GetLabel() != "@" {
		t.Fatalf("expected apex label @; got=%s", rc.GetLabel())
	}
}

func TestToRecordConfigAddsOrigin(t *testing.T) {
	rec := &record{
		RecordType: recordTypeCNAME,
		Name:       "www",
		Value:      "target.example.com",
		TTL:        300,
	}

	rc, err := toRecordConfig("example.com", rec)
	if err != nil {
		t.Fatalf("toRecordConfig returned error: %v", err)
	}
	if rc.Type != "CNAME" {
		t.Fatalf("expected CNAME; got=%s", rc.Type)
	}
	if got := rc.GetTargetField(); got != "target.example.com." {
		t.Fatalf("expected fully-qualified target with trailing dot; got=%q", got)
	}
	if rc.GetLabel() != "www" {
		t.Fatalf("expected label www; got=%s", rc.GetLabel())
	}
}

func TestToRecordConfigSRV(t *testing.T) {
	rec := &record{
		RecordType: recordTypeSRV,
		Name:       "_sip._tcp",
		Value:      "sip.example.com",
		TTL:        300,
		Priority:   u16(10),
		Weight:     u16(20),
		Port:       u16(5060),
	}

	rc, err := toRecordConfig("example.com", rec)
	if err != nil {
		t.Fatalf("toRecordConfig returned error: %v", err)
	}
	if rc.SrvPriority != 10 || rc.SrvWeight != 20 || rc.SrvPort != 5060 {
		t.Fatalf("unexpected SRV fields: %d %d %d", rc.SrvPriority, rc.SrvWeight, rc.SrvPort)
	}
}

func TestRecordTypeRoundTrip(t *testing.T) {
	for _, tc := range []string{"A", "AAAA", "CNAME", "TXT", "MX", "SRV", "CAA", "PTR", "NS"} {
		if got := recordTypeToString(recordTypeFromString(tc)); got != tc {
			t.Errorf("round trip failed for %s; got=%s", tc, got)
		}
	}
}
