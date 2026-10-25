package constellix

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func (*constellixProvider) AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2026-10-21

	a.Add("MX", rejectif.MxNull) // Last verified 2026-10-21

	a.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2026-10-21

	a.Add("NS", nsTargetIsZoneApex) // Last verified 2026-10-21

	a.Add("CAA", caaTagIsUnsupported) // Last verified 2026-10-21

	return a.Audit(records)
}

// caaTagIsUnsupported detects CAA tags other than issue, issuewild and iodef,
// which the API refuses.
func caaTagIsUnsupported(rc *models.RecordConfig) error {
	switch tag := rc.AsCAA().Tag; tag {
	case "issue", "issuewild", "iodef":
		return nil
	default:
		return fmt.Errorf("caa tag %q is not supported (only issue, issuewild and iodef)", tag)
	}
}

// nsTargetIsZoneApex detects an NS record whose target is the zone apex. The
// API answers such a record with HTTP 500 ("Server error: `POST
// .../records/ns` resulted in a `500 Internal Server Error` response"), even
// when the apex has an A record.
func nsTargetIsZoneApex(rc *models.RecordConfig) error {
	apex := rc.GetLabelFQDN()
	if label := rc.GetLabel(); label != "@" {
		apex = strings.TrimPrefix(apex, label+".")
	}
	if strings.EqualFold(strings.TrimSuffix(rc.AsNS().Ns, "."), apex) {
		return errors.New("ns target is the zone apex")
	}
	return nil
}
