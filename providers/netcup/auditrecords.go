package netcup

import (
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull)                 // Last verified 2020-12-28
	a.Add("TXT", rejectif.TxtIsEmpty)            // Last verified 2021-03-01
	a.Add("TXT", rejectif.TxtHasBackslash)       // Last verified 2026-08-04 -- Netcup API seems to strip backslashes
	a.Add("TXT", rejectif.TxtHasSingleQuotes)    // Last verified 2026-08-04 -- Netcup API seems to modify single quotes
	a.Add("TXT", rejectif.TxtHasDoubleQuotes)    // Last verified 2026-08-04 -- Netcup API seems to reject double quotes
	a.Add("CAA", rejectif.CaaTargetHasSemicolon) // Last verified 2026-08-04 -- Netcup API seems to reject semicolons in CAA target

	return a.Audit(records)
}
