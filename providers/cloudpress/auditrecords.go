package cloudpress

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	// Last verified 2026-06-22 against https://my.cloudpress.com
	a.Add("TXT", rejectif.TxtIsEmpty) // API rejects an empty TXT value.

	// CloudPress silently drops interior double quotes from TXT values
	// (e.g. `in"side` is stored as `inside`), so reject them.
	a.Add("TXT", rejectif.TxtHasDoubleQuotes)

	// CloudPress strips leading and trailing whitespace from TXT values
	// (e.g. `trailingws ` is stored as `trailingws`), so reject them.
	a.Add("TXT", rejectif.TxtStartsOrEndsWithSpaces)

	// CloudPress manages apex NS records itself and rejects custom ones:
	// "NS records are not supported on the root of the domain."
	a.Add("NS", rejectif.NsAtApex)

	return a.Audit(records)
}
