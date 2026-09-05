package infoblox

import (
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are
// supported, an empty list is returned.
func AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull)

	a.Add("NS", rejectif.NsAtApex)

	a.Add("SRV", rejectif.SrvHasNullTarget)

	a.Add("TXT", rejectif.TxtHasBackslash)

	a.Add("TXT", rejectif.TxtLongerThan(255))

	a.Add("TXT", rejectif.TxtIsEmpty)

	return a.Audit(records)
}
