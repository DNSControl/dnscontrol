package akamaiedgedns

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}
	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2023-12-12
	a.Add("TXT", rejectif.TxtHasBackslash)    // Last verified 2023-12-12

	return a.Audit(records)
}
