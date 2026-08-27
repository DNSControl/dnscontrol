package namedotcom

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

	a.Add("MX", rejectif.MxNull) // Last verified 2026-07-21

	a.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2026-07-21

	a.Add("TXT", MaxLengthNDC) // Last verified 2026-07-21

	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2026-07-21

	a.Add("TXT", rejectif.TxtHasTrailingSpace) // Last verified 2026-07-21

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2026-07-21

	return a.Audit(records)
}

// MaxLengthNDC has a length limit on TXT records. The limit is undocumented,
// but seems to be based on the encoded string, not the raw bytes.
// This seems to work.
func MaxLengthNDC(rc *models.RecordConfig) error {
	if len(rc.GetRDATA().String()) > 512 {
		return errors.New("encoded txt too long")
	}
	return nil
}
