package bunnydns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestFromRecordConfigPullZone(t *testing.T) {
	rc := &models.RecordConfig{
		Type: "BUNNY_DNS_PZ",
	}
	rc.SetLabelFromFQDN("cdn.example.com", "example.com")
	rc.MustSetTarget("12345")

	rec, err := fromRecordConfig(rc)
	if err != nil {
		t.Fatalf("fromRecordConfig returned error: %v", err)
	}
	if rec.PullZoneID != 12345 {
		t.Fatalf("expected PullZoneId=12345; got=%d", rec.PullZoneID)
	}
}

func TestFromRecordConfigPullZoneInvalidTarget(t *testing.T) {
	rc := &models.RecordConfig{
		Type: "BUNNY_DNS_PZ",
	}
	rc.SetLabelFromFQDN("cdn.example.com", "example.com")
	rc.MustSetTarget("abc")

	_, err := fromRecordConfig(rc)
	if err == nil {
		t.Fatalf("expected error for invalid Pull Zone ID")
	}
}

func TestToRecordConfigPullZoneLinkName(t *testing.T) {
	rec := &record{
		Type:     recordTypePullZone,
		Name:     "cdn",
		TTL:      300,
		LinkName: "12345",
	}

	rc, err := toRecordConfig("example.com", rec)
	if err != nil {
		t.Fatalf("toRecordConfig returned error: %v", err)
	}
	if rc.Type != "BUNNY_DNS_PZ" {
		t.Fatalf("expected type BUNNY_DNS_PZ; got=%s", rc.Type)
	}
	if rc.GetTargetField() != "12345" {
		t.Fatalf("expected target 12345; got=%s", rc.GetTargetField())
	}
	if rc.GetLabel() != "cdn" {
		t.Fatalf("expected label cdn; got=%s", rc.GetLabel())
	}
}

func TestToRecordConfigPullZoneMissingID(t *testing.T) {
	rec := &record{
		Type: recordTypePullZone,
		Name: "cdn",
		TTL:  300,
	}

	_, err := toRecordConfig("example.com", rec)
	if err == nil {
		t.Fatalf("expected error for missing Pull Zone LinkName")
	}
}
