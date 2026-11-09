package constellix

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestAuditRecords(t *testing.T) {
	dc := models.MustNewDomainConfig(testZone)

	rejected := map[string]*models.RecordConfig{
		"empty TXT":       dc.MustNewRecordConfig("foo", 300, dnsv2.TypeTXT, ""),
		"null MX":         dc.MustNewRecordConfig("foo", 300, dnsv2.TypeMX, uint16(0), "."),
		"null SRV target": dc.MustNewRecordConfig("_sip._tcp", 300, dnsv2.TypeSRV, uint16(0), uint16(0), uint16(1), "."),
		"CAA issuemail":   dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(0), "issuemail", "example.net"),
		"NS to the apex":  dc.MustNewRecordConfig("foo", 300, dnsv2.TypeNS, "example.com."),
		"NS to @":         dc.MustNewRecordConfig("a.b", 300, dnsv2.TypeNS, "@"),
	}
	for name, rc := range rejected {
		if errs := new(constellixProvider).AuditRecords(models.Records{rc}); len(errs) != 1 {
			t.Errorf("%s: AuditRecords() = %v, want one error", name, errs)
		}
	}

	accepted := models.Records{
		dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(0), "issue", "letsencrypt.org"),
		dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(0), "issuewild", ";"),
		dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(128), "iodef", "mailto:a@example.net"),
		dc.MustNewRecordConfig("foo", 300, dnsv2.TypeTXT, `say "hi" \ café`),
		dc.MustNewRecordConfig("foo", 300, dnsv2.TypeMX, uint16(10), "mx.example.net."),
		dc.MustNewRecordConfig("foo", 300, dnsv2.TypeNS, "ns.example.com."),
		dc.MustNewRecordConfig("@", 300, dnsv2.TypeNS, "ns1.example.net."),
	}
	if errs := new(constellixProvider).AuditRecords(accepted); len(errs) != 0 {
		t.Errorf("AuditRecords() = %v, want none", errs)
	}
}
