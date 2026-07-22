package cloudns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestToRcConvertsCloudWRToCloudnsWR(t *testing.T) {
	// ClouDNS API returns "CLOUD_WR" as the type for web redirect records.
	// dnscontrol uses "CLOUDNS_WR" as the custom record type.
	// Verify that toRc maps "CLOUD_WR" -> "CLOUDNS_WR" so that fetched
	// records match desired records and are not destroyed/recreated every push.
	r := &domainRecord{
		ID:     "123",
		Type:   "CLOUD_WR",
		Host:   "www",
		Target: "https://example.com",
		TTL:    "3600",
	}

	dc := models.MustNewDomainConfig("example.com")
	rc, err := toRc(dc, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.Type != "CLOUDNS_WR" {
		t.Errorf("expected type CLOUDNS_WR, got %s", rc.Type)
	}
	if rc.GetTargetField() != r.Target {
		t.Errorf("expected target %q, got %q", r.Target, rc.GetTargetField())
	}
}

func TestToRcMX(t *testing.T) {
	r := &domainRecord{
		Type:     "MX",
		Host:     "www",
		Target:   "mail.example.net",
		Priority: "10",
		TTL:      "3600",
	}

	dc := models.MustNewDomainConfig("example.com")
	rc, err := toRc(dc, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := rc.GetRDATA().String(); got != "10 mail.example.net." {
		t.Errorf("expected MX data %q, got %q", "10 mail.example.net.", got)
	}
}
