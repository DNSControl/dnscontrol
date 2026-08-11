package gidinet

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are supported,
// an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull) // Last verified 2026-01-24

	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2026-01-24

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2026-01-24

	a.Add("TXT", rejectif.TxtHasBackticks) // Last verified 2026-01-24

	a.Add("TXT", rejectif.TxtHasBackslash) // Last verified 2026-01-25

	// Note: Long TXT records (>250 chars) are supported via automatic chunking.
	// The API accepts format: "chunk1" "chunk2" where each chunk is ≤250 chars.

	return a.Audit(records)
}
