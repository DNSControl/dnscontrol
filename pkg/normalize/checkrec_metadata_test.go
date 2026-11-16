package normalize

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestCheckObsoleteMetadata(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rec := dc.MustNewRecordConfig("www", 0, dnsv2.TypeA, "192.0.2.1")
	if err := checkObsoleteMetadata(rec); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	rec.Metadata["ignore_name_disable_safety_check"] = "true"
	if err := checkObsoleteMetadata(rec); err == nil {
		t.Error("expected an error for ignore_name_disable_safety_check, got none")
	}
}
