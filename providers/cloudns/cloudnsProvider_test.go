package cloudns

import (
	"testing"
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

	rc, err := toRc("example.com", r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.Type != "CLOUDNS_WR" {
		t.Errorf("expected type CLOUDNS_WR, got %s", rc.Type)
	}
}
