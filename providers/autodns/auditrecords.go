package autodns

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull)      // Last verified 2022-03-25
	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2025-05-13

	// Last verified 2026-06-22: AutoDNS drops interior double quotes from TXT
	// values (e.g. `in"side` is stored as `inside`).
	a.Add("TXT", rejectif.TxtHasDoubleQuotes)

	// Last verified 2026-06-22: AutoDNS strips an odd (unpaired) run of
	// backslashes from TXT values (e.g. `1back\slash` is stored as
	// `1backslash`); even runs are preserved.
	a.Add("TXT", rejectif.TxtHasUnpairedBackslash)

	// Last verified 2026-06-22: AutoDNS trims surrounding whitespace from TXT
	// values (e.g. a trailing space in `trailingws ` is stripped).
	a.Add("TXT", rejectif.TxtStartsOrEndsWithSpaces)

	return a.Audit(records)
}
