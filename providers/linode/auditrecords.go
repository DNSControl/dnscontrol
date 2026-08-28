package linode

import (
	"errors"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.Add("CAA", rejectif.CaaFlagIsNonZero)            // Last verified 2026-07-28
	a.Add("CAA", rejectif.CaaTargetContainsWhitespace) // Last verified 2023-01-15

	a.Add("SRV", srvHasInvalidLabel) // Last verified 2026-08-28

	return a.Audit(records)
}

// srvHasInvalidLabel rejects SRV records whose label is not of the form
// "_service._protocol" (optionally followed by a subdomain). Linode stores
// SRV records by their service and protocol, so a malformed label cannot be
// represented.
func srvHasInvalidLabel(rc *models.RecordConfig) error {
	if !srvLabelRegexp.MatchString(rc.GetLabel()) {
		return errors.New(`SRV label must match format "_service._protocol"`)
	}
	return nil
}
