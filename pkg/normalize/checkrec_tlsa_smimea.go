package normalize

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// checkTLSASMIMEA checks that the Usage, Selector and MatchingType of a TLSA
// or SMIMEA record are in range. Other record types are not checked.
func checkTLSASMIMEA(rec *models.RecordConfig, domain string) (errs []error) {
	switch rec.Type {
	case "TLSA":
		f := rec.AsTLSA()
		if f.Usage > 3 {
			errs = append(errs, fmt.Errorf("TLSA Usage %d is invalid in record %s (domain %s)",
				f.Usage, rec.GetLabel(), domain))
		}
		if f.Selector > 1 {
			errs = append(errs, fmt.Errorf("TLSA Selector %d is invalid in record %s (domain %s)",
				f.Selector, rec.GetLabel(), domain))
		}
		if f.MatchingType > 2 {
			errs = append(errs, fmt.Errorf("TLSA MatchingType %d is invalid in record %s (domain %s)",
				f.MatchingType, rec.GetLabel(), domain))
		}
	case "SMIMEA":
		f := rec.AsSMIMEA()
		if f.Usage > 3 {
			errs = append(errs, fmt.Errorf("SMIMEA Usage %d is invalid in record %s (domain %s)",
				f.Usage, rec.GetLabel(), domain))
		}
		if f.Selector > 1 {
			errs = append(errs, fmt.Errorf("SMIMEA Selector %d is invalid in record %s (domain %s)",
				f.Selector, rec.GetLabel(), domain))
		}
		if f.MatchingType > 2 {
			errs = append(errs, fmt.Errorf("SMIMEA MatchingType %d is invalid in record %s (domain %s)",
				f.MatchingType, rec.GetLabel(), domain))
		}
	}
	return errs
}
