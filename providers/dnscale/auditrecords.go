package dnscale

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull) // MX records must have a target

	a.Add("TXT", rejectif.TxtHasDoubleQuotes)        // TXT records shouldn't contain unescaped double quotes
	a.Add("TXT", rejectif.TxtIsEmpty)                // DNScale doesn't support empty TXT records
	a.Add("TXT", rejectif.TxtStartsOrEndsWithSpaces) // DNScale doesn't support leading/trailing whitespace in TXT records

	return a.Audit(records)
}
