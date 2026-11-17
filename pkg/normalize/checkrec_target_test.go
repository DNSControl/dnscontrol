package normalize

import (
	"fmt"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestSoaLabelAndTarget(t *testing.T) {
	tests := []struct {
		isError bool
		label   string
		target  string
	}{
		{false, "@", "ns1.foo.com."},

		// Invalid target
		// {true, "@", "ns1.foo.com"},
		// Commented out because MustNewRecordConfig fixes this.

		// Invalid label, only '@' is allowed for SOA records
		{true, "foo.com", "ns1.foo.com."},
	}
	dc := models.MustNewDomainConfig("foo.com")
	for _, test := range tests {
		experiment := fmt.Sprintf("%s %s", test.label, test.target)
		rc := dc.MustNewRecordConfig(test.label, 0, dnsv2.TypeSOA, test.target, "bar.foo.com.", 1, 1, 1, 1, 1)
		err := checkTargets(rc, "foo.com")
		if err != nil && !test.isError {
			t.Errorf("%v: Error (%v)\n", experiment, err)
		}
		if err == nil && test.isError {
			t.Errorf("%v: Expected error but got none \n", experiment)
		}
	}
}

func TestNSAtRoot(t *testing.T) {
	dc := models.MustNewDomainConfig("foo.com")
	// do not allow ns records for @
	rec := dc.MustNewRecordConfig(dc.LabelFromShort("test"), 0, dnsv2.TypeNS, "ns1.name.com.")
	errs := checkTargets(rec, "foo.com")
	if len(errs) > 0 {
		t.Error("Expect no error with ns record on subdomain")
	}
	rec.SetLabel("@", "foo.com")
	errs = checkTargets(rec, "foo.com")
	if len(errs) != 1 {
		t.Error("Expect error with ns record on @")
	}
}
