package normalize

import (
	"errors"
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nameutil"
)

// checkTargets returns zero or more errors when problems are found.
// Checks of the RDATA itself are done by the Make*() functions. What remains
// here are checks that depend on the record's label.
func checkTargets(rec *models.RecordConfig, domain string) (errs []error) {
	label := rec.GetLabel()
	check := func(e error) {
		if e != nil {
			err := fmt.Errorf("%s: %s %s: %s", rec.FilePos, rec.Type, rec.GetLabelFQDN(), e.Error())
			if _, ok := e.(Warning); ok {
				err = Warning{err}
			}
			errs = append(errs, err)
		}
	}
	switch rec.Type { // #rtype_variations

	// IMPORT_TRANSFORM is not in the record catalog, so the default case would reject it.
	case "IMPORT_TRANSFORM":

	case "CNAME":
		if label == "@" {
			check(errors.New("cannot create CNAME record for bare domain. Use ALIAS"))
		}
		labelFQDN := nameutil.ToFqdnNoDot(label, domain)
		targetFQDN := nameutil.ToFqdnNoDot(rec.AsCNAME().Target, domain)
		if labelFQDN == targetFQDN {
			check(errors.New("CNAME loop (target points at itself)"))
		}
	case "NS":
		if label == "@" {
			check(errors.New("cannot create NS record for bare domain. Use NAMESERVER instead"))
		}
	case "SOA":
		if label != "@" {
			check(errors.New("SOA record is only valid for bare domain"))
		}
	}
	return errs
}
