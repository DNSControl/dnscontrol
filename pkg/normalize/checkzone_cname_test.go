package normalize

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestCNAMECloudflareProxied(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	// A proxied (flattened) CNAME should be allowed alongside other record types.
	recCNAME := dc.MustNewRecordConfig("mail", 0, dnsv2.TypeCNAME, "example.com.")
	recCNAME.Metadata["cloudflare_proxy"] = "on"
	dc.AddRecordConfig(recCNAME)
	recMX := dc.MustNewRecordConfig("mail", 0, dnsv2.TypeMX, 10, "smtp.example.com.")
	dc.AddRecordConfig(recMX)
	errs := checkCNAMEs(dc, nil)
	if len(errs) != 0 {
		t.Errorf("Expected no errors for proxied CNAME + MX, got: %v", errs)
	}

	// A non-proxied CNAME should still fail.
	dc2 := models.MustNewDomainConfig("example.com")
	recCNAME2 := dc2.MustNewRecordConfig("mail", 0, dnsv2.TypeCNAME, "example.com.")
	//recCNAME2.SetLabel("mail", "mail.example.com")
	recMX2 := dc2.MustNewRecordConfig("mail", 0, dnsv2.TypeMX, 10, "smtp.example.com.")
	dc2.AddRecordConfig(recCNAME2)
	dc2.AddRecordConfig(recMX2)
	errs2 := checkCNAMEs(dc2, nil)
	if len(errs2) == 0 {
		t.Error("Expected error for non-proxied CNAME + MX, got none")
	}
}
