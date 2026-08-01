package ovh

import (
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("TXT", rejectif.TxtHasBackslash) // Last verified 2026-07-19

	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2026-07-19

	a.Add("TXT", rejectif.TxtStartsOrEndsWithSpaces) // Last verified 2026-08-01

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2026-07-26
	a.Add("TXT", func(rc *models.RecordConfig) error {
		if rc.Type == "DKIM" && !strings.HasSuffix(rc.Name, "._domainkey") {
			return errors.New("DKIM name should end with ._domainkey")
		}
		return nil
	}) // Last verified 2026-07-26

	return a.Audit(records)
}
