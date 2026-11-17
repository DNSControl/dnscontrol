package rejectif

import (
	"github.com/DNSControl/dnscontrol/v5/models"
)

// Auditor stores a list of checks to be executed during Audit().
type Auditor struct {
	checksFor map[string][]checker
}

type checker = func(*models.RecordConfig) error

// Add registers a function to call on each record of a given type.
func (aud *Auditor) Add(rtype string, fn checker) {
	if aud.checksFor == nil {
		aud.checksFor = map[string][]checker{}
	}
	aud.checksFor[rtype] = append(aud.checksFor[rtype], fn)

	// SPF records get any checkers that TXT records do.
	if rtype == "TXT" {
		aud.Add("SPF", fn)
	}
}

// Audit performs the audit. For each record it calls each function in
// the list of checks.
func (aud *Auditor) Audit(records models.Records) (errs []error) {
	// No checks? Exit early.
	if aud.checksFor == nil {
		return nil
	}

	// For each record, call the checks for that type, gather errors.
	for _, rc := range records {
		// First, run type-specific checks
		for _, f := range aud.checksFor[rc.Type] {
			e := f(rc)
			if e != nil {
				errs = append(errs, e)
			}
		}
		// Then, run wildcard checks that apply to all record types
		for _, f := range aud.checksFor["*"] {
			e := f(rc)
			if e != nil {
				errs = append(errs, e)
			}
		}
	}

	return errs
}
