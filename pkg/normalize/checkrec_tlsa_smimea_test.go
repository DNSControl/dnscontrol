package normalize

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestTLSAValidation(t *testing.T) {
	dc := models.MustNewDomainConfig("_443._tcp.example.com")
	dc.RegistrarName = "BIND"
	dc.AddTestRC(t, "_443._tcp", 0, dnsv2.TypeTLSA, 4, 1, 1, "abcdef0")

	config := &models.DNSConfig{}
	config.Domains = append(config.Domains, dc)

	errs := ValidateAndNormalizeConfig(config)
	if len(errs) != 1 {
		t.Error("Expect error on invalid TLSA but got none")
	}
}

func TestCheckTLSASMIMEA(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name         string
		rtype        uint16
		usage        int
		selector     int
		matchingType int
		wantErrs     int
	}{
		{"TLSA valid", dnsv2.TypeTLSA, 3, 1, 1, 0},
		{"TLSA max values", dnsv2.TypeTLSA, 3, 1, 2, 0},
		{"TLSA bad usage", dnsv2.TypeTLSA, 4, 1, 1, 1},
		{"TLSA bad selector", dnsv2.TypeTLSA, 3, 2, 1, 1},
		{"TLSA bad matching type", dnsv2.TypeTLSA, 3, 1, 3, 1},
		{"TLSA all bad", dnsv2.TypeTLSA, 4, 2, 3, 3},
		{"SMIMEA valid", dnsv2.TypeSMIMEA, 3, 1, 1, 0},
		{"SMIMEA bad usage", dnsv2.TypeSMIMEA, 4, 1, 1, 1},
		{"SMIMEA bad selector", dnsv2.TypeSMIMEA, 3, 2, 1, 1},
		{"SMIMEA bad matching type", dnsv2.TypeSMIMEA, 3, 1, 3, 1},
		{"SMIMEA all bad", dnsv2.TypeSMIMEA, 4, 2, 3, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := dc.MustNewRecordConfig("_443._tcp", 0, tt.rtype, tt.usage, tt.selector, tt.matchingType, "abcdef01")
			errs := checkTLSASMIMEA(rec, dc.Name)
			if len(errs) != tt.wantErrs {
				t.Errorf("checkTLSASMIMEA() returned %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}

func TestCheckTLSASMIMEA_otherTypes(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rec := dc.MustNewRecordConfig("www", 0, dnsv2.TypeA, "1.2.3.4")
	if errs := checkTLSASMIMEA(rec, dc.Name); len(errs) != 0 {
		t.Errorf("checkTLSASMIMEA() on an A record returned errors: %v", errs)
	}
}
