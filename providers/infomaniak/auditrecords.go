package infomaniak

import (
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are
// supported, an empty list is returned.
func AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull) // Last verified 2026-07-26

	a.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2026-07-26

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2026-07-26

	return a.Audit(records)
}
