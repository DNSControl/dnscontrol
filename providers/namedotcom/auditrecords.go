package namedotcom

import (
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull) // Last verified 2020-12-28

	a.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2020-12-28

	a.Add("TXT", MaxLengthNDC) // Last verified 2024-12-17

	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2024-12-17

	a.Add("TXT", rejectif.TxtHasTrailingSpace) // Last verified 2024-12-17

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2024-12-17

	return a.Audit(records)
}

// MaxLengthNDC returns and error if the sum of the strings
// are longer than permitted by NDC. Sadly their
// length limit is undocumented. This seems to work.
func MaxLengthNDC(rc *models.RecordConfig) error {
	txtStrings := rc.GetTargetTXTSegmented()
	if len(txtStrings) == 0 {
		return nil
	}

	sum := 2 // Count the start and end quote.
	// Add the length of each segment.
	for _, segment := range txtStrings {
		sum += len(segment)                // The length of each segment
		sum += strings.Count(segment, `"`) // Add 1 for any char to be escaped
	}
	// Add 3 (quote space quote) for each interior join.
	sum += 3 * (len(txtStrings) - 1)

	if sum > 512 {
		return errors.New("encoded txt too long")
	}
	return nil
}
