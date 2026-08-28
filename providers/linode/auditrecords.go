package linode

import (
	"errors"
	"regexp"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// srvLabelRegexp matches a valid SRV record label: "_service._protocol",
// optionally followed by a subdomain (e.g. "_smtp._tcp.sub.domain").
//
// Each of the service and protocol tokens is a single leading underscore
// followed by an alphanumeric and then any number of alphanumerics or hyphens
// (no additional underscores). The named Service and Protocol capture groups
// let linodeProvider.go extract those two values, so this single regexp is used
// both to validate the label (AuditRecords) and to parse it (toReq).
var srvLabelRegexp = regexp.MustCompile(`^_(?P<Service>[A-Za-z0-9][A-Za-z0-9-]*)\._(?P<Protocol>[A-Za-z0-9][A-Za-z0-9-]*)(\.[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?)*$`)

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
