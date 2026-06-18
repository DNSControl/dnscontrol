package websupport

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2026-06-17

	a.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2026-06-17

	// The WebSupport v2 DNS record API silently ignores attempts to create NS
	// records (it returns success but the record never appears), which would
	// otherwise cause dnscontrol to loop trying to create them. Reject them
	// with a clear message instead. Last verified 2026-06-17.
	a.Add("NS", rejectNS)

	return a.Audit(records)
}

func rejectNS(rc *models.RecordConfig) error {
	return fmt.Errorf("WEBSUPPORT does not support managing NS records via its API")
}
