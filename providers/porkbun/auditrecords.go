package porkbun

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2022-11-19

	a.Add("TXT", rejectif.TxtHasTrailingSpace) // Last verified 2022-11-19

	a.Add("CAA", rejectif.CaaTargetContainsWhitespace) // Last verified 2024-11-11

	return a.Audit(records)
}
