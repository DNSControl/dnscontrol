package loopia

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("CAA", rejectif.CaaTargetContainsWhitespace) // Last verified 2025-07-24: Loopia returns 404

	a.Add("MX", rejectif.MxNull) // Last verified 2025-07-24: Loopia returns 404

	a.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2025-07-24: Loopia returns 404

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2025-07-24: Loopia returns 404

	a.Add("TXT", rejectif.TxtLongerThan(450)) // Last verified 2025-07-24: Loopia returns 404

	return a.Audit(records)
}
