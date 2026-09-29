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

func TestFromRecordConfigGeographicRouting(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("www", 300, "A", "1.2.3.4")
	rc.Metadata = map[string]string{
		metaSmartRoutingType:     "geographic",
		metaGeolocationLatitude:  "40.7128",
		metaGeolocationLongitude: "-74.0060",
	}

	rec, err := fromRecordConfig(rc)
	if err != nil {
		t.Fatalf("fromRecordConfig returned error: %v", err)
	}
	if rec.SmartRoutingType != smartRoutingGeographic {
		t.Fatalf("expected SmartRoutingType=%d; got=%d", smartRoutingGeographic, rec.SmartRoutingType)
	}
	if rec.GeolocationLatitude == nil || *rec.GeolocationLatitude != 40.7128 {
		t.Fatalf("expected GeolocationLatitude=40.7128; got=%v", rec.GeolocationLatitude)
	}
	if rec.GeolocationLongitude == nil || *rec.GeolocationLongitude != -74.0060 {
		t.Fatalf("expected GeolocationLongitude=-74.0060; got=%v", rec.GeolocationLongitude)
	}
	if rec.LatencyZone != "" {
		t.Fatalf("expected empty LatencyZone; got=%q", rec.LatencyZone)
	}
}

func TestFromRecordConfigLatencyRouting(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("www", 300, "A", "1.2.3.4")
	rc.Metadata = map[string]string{
		metaSmartRoutingType: "latency",
		metaLatencyZone:      "NY",
	}

	rec, err := fromRecordConfig(rc)
	if err != nil {
		t.Fatalf("fromRecordConfig returned error: %v", err)
	}
	if rec.SmartRoutingType != smartRoutingLatency {
		t.Fatalf("expected SmartRoutingType=%d; got=%d", smartRoutingLatency, rec.SmartRoutingType)
	}
	if rec.LatencyZone != "NY" {
		t.Fatalf("expected LatencyZone=NY; got=%q", rec.LatencyZone)
	}
	if rec.GeolocationLatitude != nil || rec.GeolocationLongitude != nil {
		t.Fatalf("expected nil geolocation coords for latency routing")
	}
}

func TestFromRecordConfigInvalidSmartRoutingType(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("www", 300, "A", "1.2.3.4")
	rc.Metadata = map[string]string{metaSmartRoutingType: "invalid"}

	_, err := fromRecordConfig(rc)
	if err == nil {
		t.Fatalf("expected error for invalid smart routing type")
	}
}

func TestFromRecordConfigInvalidLatitude(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("www", 300, "A", "1.2.3.4")
	rc.Metadata = map[string]string{
		metaSmartRoutingType:    "geographic",
		metaGeolocationLatitude: "not-a-number",
	}

	_, err := fromRecordConfig(rc)
	if err == nil {
		t.Fatalf("expected error for invalid latitude")
	}
}

func TestFromRecordConfigSmartRoutingOnlyOnAAndAAAA(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("www", 300, "CNAME", "target.example.com.")
	rc.Metadata = map[string]string{
		metaSmartRoutingType: "latency",
		metaLatencyZone:      "NY",
	}

	rec, err := fromRecordConfig(rc)
	if err != nil {
		t.Fatalf("fromRecordConfig returned error: %v", err)
	}
	if rec.SmartRoutingType != smartRoutingNone {
		t.Fatalf("expected SmartRoutingType=0 for non-A/AAAA; got=%d", rec.SmartRoutingType)
	}
	if rec.LatencyZone != "" {
		t.Fatalf("expected empty LatencyZone for non-A/AAAA; got=%q", rec.LatencyZone)
	}
}

func TestToRecordConfigGeographicRouting(t *testing.T) {
	lat := 40.7128
	lon := -74.0060
	rec := &record{
		Type:                 recordTypeA,
		Name:                 "www",
		Value:                "1.2.3.4",
		TTL:                  300,
		SmartRoutingType:     smartRoutingGeographic,
		GeolocationLatitude:  &lat,
		GeolocationLongitude: &lon,
	}

	rc, err := toRecordConfig(models.MustNewDomainConfig("example.com"), rec)
	if err != nil {
		t.Fatalf("toRecordConfig returned error: %v", err)
	}
	if rc.Metadata[metaSmartRoutingType] != "geographic" {
		t.Fatalf("expected metadata %s=geographic; got=%q", metaSmartRoutingType, rc.Metadata[metaSmartRoutingType])
	}
	if rc.Metadata[metaGeolocationLatitude] != "40.7128" {
		t.Fatalf("expected metadata %s=40.7128; got=%q", metaGeolocationLatitude, rc.Metadata[metaGeolocationLatitude])
	}
	if rc.Metadata[metaGeolocationLongitude] != "-74.006" {
		t.Fatalf("expected metadata %s=-74.006; got=%q", metaGeolocationLongitude, rc.Metadata[metaGeolocationLongitude])
	}
	if _, ok := rc.Metadata[metaLatencyZone]; ok {
		t.Fatalf("expected no %s metadata for geographic routing", metaLatencyZone)
	}
}

func TestToRecordConfigLatencyRouting(t *testing.T) {
	rec := &record{
		Type:             recordTypeAAAA,
		Name:             "www",
		Value:            "::1",
		TTL:              300,
		SmartRoutingType: smartRoutingLatency,
		LatencyZone:      "NY",
	}

	rc, err := toRecordConfig(models.MustNewDomainConfig("example.com"), rec)
	if err != nil {
		t.Fatalf("toRecordConfig returned error: %v", err)
	}
	if rc.Metadata[metaSmartRoutingType] != "latency" {
		t.Fatalf("expected metadata %s=latency; got=%q", metaSmartRoutingType, rc.Metadata[metaSmartRoutingType])
	}
	if rc.Metadata[metaLatencyZone] != "NY" {
		t.Fatalf("expected metadata %s=NY; got=%q", metaLatencyZone, rc.Metadata[metaLatencyZone])
	}
	if _, ok := rc.Metadata[metaGeolocationLatitude]; ok {
		t.Fatalf("expected no %s metadata for latency routing", metaGeolocationLatitude)
	}
}

func TestToRecordConfigNoSmartRouting(t *testing.T) {
	rec := &record{
		Type:  recordTypeA,
		Name:  "www",
		Value: "1.2.3.4",
		TTL:   300,
	}

	rc, err := toRecordConfig(models.MustNewDomainConfig("example.com"), rec)
	if err != nil {
		t.Fatalf("toRecordConfig returned error: %v", err)
	}
	if len(rc.Metadata) != 0 {
		t.Fatalf("expected no metadata for record without smart routing; got=%v", rc.Metadata)
	}
}

func TestParseSmartRoutingType(t *testing.T) {
	tests := []struct {
		input string
		want  smartRoutingType
		err   bool
	}{
		{"", smartRoutingNone, false},
		{"none", smartRoutingNone, false},
		{"latency", smartRoutingLatency, false},
		{"LATENCY", smartRoutingLatency, false},
		{"geographic", smartRoutingGeographic, false},
		{"geo", smartRoutingGeographic, false},
		{"GEO", smartRoutingGeographic, false},
		{"invalid", smartRoutingNone, true},
	}
	for _, tt := range tests {
		got, err := parseSmartRoutingType(tt.input)
		if tt.err {
			if err == nil {
				t.Errorf("parseSmartRoutingType(%q) expected error; got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSmartRoutingType(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseSmartRoutingType(%q) expected %d; got %d", tt.input, tt.want, got)
		}
	}
}
