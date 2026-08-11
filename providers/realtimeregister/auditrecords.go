package realtimeregister

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	auditor := rejectif.Auditor{}

	auditor.Add("TXT", rejectif.TxtHasTrailingSpace) // Last verified 2024-01-03

	auditor.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2024-01-03

	return auditor.Audit(records)
}
