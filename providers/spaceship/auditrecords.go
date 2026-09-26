package spaceship

import (
	"errors"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns errors for records this provider cannot represent.
func AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.Add("TXT", rejectif.TxtIsEmpty)
	a.Add("TXT", rejectif.TxtHasTrailingSpace)
	a.Add("CAA", rejectif.CaaTargetContainsWhitespace)
	a.Add("MX", rejectif.MxNull)            // Last verified 2026-09-20: API 422 "Exchange field is required"
	a.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2026-09-20: API 422 "Target field is required"
	a.Add("NS", rejectif.NsAtApex)
	a.Add("ALIAS", func(rc *models.RecordConfig) error {
		if rc.GetLabel() == "@" {
			return errors.New("apex ALIAS is stored as CNAME; declare an apex CNAME instead")
		}
		return nil
	})

	return a.Audit(records)
}
