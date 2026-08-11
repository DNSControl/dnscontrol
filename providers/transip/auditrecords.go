package transip

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("ALIAS", rejectif.LabelNotApex) // Last verified 2024-01-11

	a.Add("MX", rejectif.MxNull) // Last verified 2023-12-04

	a.Add("TXT", rejectif.TxtHasBackticks) // Last verified 2024-01-11

	a.Add("TXT", rejectif.TxtHasBackslash) // Last verified 2024-01-11

	a.Add("TXT", rejectif.TxtStartsOrEndsWithSpaces) // Last verified 2024-01-11

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2024-01-11

	a.Add("TXT", rejectif.TxtLongerThan(1024)) // Last verified 2024-01-11

	a.Add("TXT", rejectif.TxtHasTrailingSpace) // Last verified 2024-01-11

	return a.Audit(records)
}
