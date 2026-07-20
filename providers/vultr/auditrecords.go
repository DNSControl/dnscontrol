package vultr

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull) // Last verified 2020-12-28

	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2021-03-02
	// Needs investigation. Could be a dnscontrol issue or
	// the provider doesn't support double quotes.

	a.Add("CAA", rejectif.CaaTargetContainsWhitespace) // Last verified 2023-01-19

	return a.Audit(records)
}
