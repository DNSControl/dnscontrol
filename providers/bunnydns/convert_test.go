package bunnydns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
)

func TestFromRecordConfigPullZone(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("cdn", 0, privatetypes.TypeBUNNYDNSPZ, int64(12345))

	rec, err := fromRecordConfig(rc)
	if err != nil {
		t.Fatalf("fromRecordConfig returned error: %v", err)
	}
	if rec.PullZoneID != 12345 {
		t.Fatalf("expected PullZoneId=12345; got=%d", rec.PullZoneID)
	}
}

func TestFromRecordConfigRedirect(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("go", 0, privatetypes.TypeBUNNYDNSRDR, "https://example.com")

	rec, err := fromRecordConfig(rc)
	if err != nil {
		t.Fatalf("fromRecordConfig returned error: %v", err)
	}
	if rec.Value != "https://example.com" {
		t.Fatalf("expected redirect value; got=%q", rec.Value)
	}
}

func TestToRecordConfigRedirect(t *testing.T) {
	rec := &record{
		Type:  recordTypeRedirect,
		Name:  "go",
		TTL:   300,
		Value: "https://example.com",
	}

	dc := models.MustNewDomainConfig("example.com")
	rc, err := toRecordConfig(dc, rec)
	if err != nil {
		t.Fatalf("toRecordConfig returned error: %v", err)
	}
	if got := rc.AsBUNNYDNSRDR().Target; got != "https://example.com" {
		t.Fatalf("redirect target = %q, want https://example.com", got)
	}
}

func TestToRecordConfigPullZoneLinkName(t *testing.T) {
	rec := &record{
		Type:     recordTypePullZone,
		Name:     "cdn",
		TTL:      300,
		LinkName: "12345",
	}

	dc := models.MustNewDomainConfig("example.com")
	rc, err := toRecordConfig(dc, rec)
	if err != nil {
		t.Fatalf("toRecordConfig returned error: %v", err)
	}
	if rc.Type != "BUNNY_DNS_PZ" {
		t.Fatalf("expected type BUNNY_DNS_PZ; got=%s", rc.Type)
	}
	rdata := rc.GetRDATA().(privatetypesrdata.BUNNYDNSPZ)
	if rdata.PullZoneID != 12345 {
		t.Fatalf("expected PullZoneId=12345; got=%d", rdata.PullZoneID)
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

	dc := models.MustNewDomainConfig("example.com")
	_, err := toRecordConfig(dc, rec)
	if err == nil {
		t.Fatalf("expected error for missing Pull Zone LinkName")
	}
}
