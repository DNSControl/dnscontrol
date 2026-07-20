package joker

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

var supportedRTypes = map[string]struct{}{
	"A":     {},
	"AAAA":  {},
	"CAA":   {},
	"CNAME": {},
	"MX":    {},
	"NAPTR": {},
	"NS":    {},
	"SRV":   {},
	"TXT":   {},
}

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	// Joker does not support custom NS records at apex (domain root)
	// Joker automatically manages apex NS records
	a.Add("NS", rejectif.NsAtApex)

	// Joker has round-trip issues with TXT records containing unbalanced quotes
	a.Add("TXT", rejectif.TxtHasUnpairedDoubleQuotes)

	// Joker has round-trip issues with TXT records containing backslashes
	a.Add("TXT", rejectif.TxtHasBackslash)

	// Joker has round-trip issues with TXT records containing backslashes
	a.Add("TXT", rejectif.TxtHasTrailingSpace)

	// SRV records must have valid port and target
	a.Add("SRV", rejectif.SrvHasZeroPort)
	a.Add("SRV", rejectif.SrvHasEmptyTarget)

	// CAA records must have valid tag and target
	a.Add("CAA", rejectif.CaaHasEmptyTag)
	a.Add("CAA", rejectif.CaaHasEmptyTarget)

	// NAPTR records must have a replacement
	a.Add("NAPTR", rejectif.NaptrHasEmptyTarget)

	errors := a.Audit(records)

	// Check for unsupported record types
	for _, r := range records {
		if _, ok := supportedRTypes[r.Type]; !ok {
			errors = append(errors, fmt.Errorf("joker does not support %s records", r.Type))
		}
	}

	return errors
}
