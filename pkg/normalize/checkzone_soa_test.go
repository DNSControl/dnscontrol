package normalize

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestCheckMultipleSOAs(t *testing.T) {
	dcSingleSoa := models.MustNewDomainConfig("foo.com")
	dcSingleSoa.RegistrarName = "BIND"
	dcSingleSoa.AddTestRC(t, "@", 0, dnsv2.TypeSOA, "ns1.foo.com.", "bar.foo.com", 1, 1, 1, 1, 1)

	t.Run("single_SOA", func(t *testing.T) {
		errs := checkMultipleSOAs(dcSingleSoa)
		if errs != nil {
			t.Error("checkMultipleSOAs function failed with single SOA record")
		}
	})

	dcTwoSoas := dcSingleSoa
	// Add another SOA record to DC
	dcTwoSoas.AddTestRC(t, "@", 0, dnsv2.TypeSOA, "ns2.foo.com.", "bar.foo.com", 1, 1, 1, 1, 1)

	t.Run("two_SOAs", func(t *testing.T) {
		errs := checkMultipleSOAs(dcTwoSoas)
		if len(errs) < 1 {
			t.Error("checkMultipleSOAs function failed to catch two SOAs")
		}
	})
}
