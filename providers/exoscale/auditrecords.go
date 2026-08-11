package exoscale

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records that aren't supported by this provider.
// If all records are supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	auditor := rejectif.Auditor{}

	auditor.Add("CAA", rejectif.CaaTargetContainsWhitespace) // Last verified 2022-07-11

	auditor.Add("MX", rejectif.MxNull) // Last verified 2022-07-11

	auditor.Add("PTR", func(rc *models.RecordConfig) error {
		return fmt.Errorf("PTR records are not supported by the Exoscale provider")
	})

	auditor.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2020-12-28

	auditor.Add("TXT", rejectif.TxtHasUnpairedBackslash) // Last verified 2026-05-04
	auditor.Add("TXT", rejectif.TxtHasDoubleQuotes)      // Last verified 2026-05-04
	auditor.Add("TXT", rejectif.TxtIsEmpty)              // Last verified 2026-05-04

	return auditor.Audit(records)
}
