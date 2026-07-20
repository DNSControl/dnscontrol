package alidns

import (
	"errors"
	"unicode"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
	"golang.org/x/net/idna"
)

// isValidAliDNSString checks if a string contains only ASCII or Chinese characters.
// Alibaba Cloud DNS allows: a-z, A-Z, 0-9, -, _, ., *, @, and Chinese characters (汉字).
func isValidAliDNSString(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			// Allow CJK Unified Ideographs (Chinese characters): U+4E00 to U+9FFF
			// and CJK Extension A: U+3400 to U+4DBF
			if (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF) {
				continue
			}
			return false
		}
	}
	return true
}

// labelConstraint detects labels that contain non-ASCII characters except Chinese characters.
func labelConstraint(rc *models.RecordConfig) error {
	if !isValidAliDNSString(rc.GetLabel()) {
		return errors.New("label contains non-ASCII characters (only Chinese is allowed)")
	}
	return nil
}

// targetConstraint detects target values that contain non-ASCII characters except Chinese characters.
// This applies to CNAME, MX, NS, SRV targets.
func targetConstraint(rc *models.RecordConfig) error {
	target, err := idna.ToUnicode(rc.GetTargetField())
	if err != nil {
		target = rc.GetTargetField()
	}
	if !isValidAliDNSString(target) {
		return errors.New("target contains non-ASCII characters (only Chinese is allowed)")
	}
	return nil
}

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	// Note: We can't get domain version info here because AuditRecords
	// is called without provider context. TTL validation will be done
	// at the provider level in GetZoneRecordsCorrections.
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull)                   // Last verified at 2025-12-03
	a.Add("TXT", rejectif.TxtIsEmpty)              // Last verified at 2025-12-03
	a.Add("TXT", rejectif.TxtLongerThan(512))      // Last verified at 2025-12-03: 511 bytes OK, 764 bytes failed
	a.Add("TXT", rejectif.TxtHasDoubleQuotes)      // Last verified at 2025-12-03: Alibaba strips quotes
	a.Add("TXT", rejectif.TxtHasTrailingSpace)     // Last verified at 2025-12-03: Alibaba strips trailing spaces
	a.Add("TXT", rejectif.TxtHasUnpairedBackslash) // Last verified at 2025-12-03: Alibaba mishandles odd backslashes
	a.Add("*", labelConstraint)                    // Last verified at 2025-12-03: Alibaba only allows ASCII + Chinese, rejects other Unicode
	a.Add("CNAME", targetConstraint)               // Last verified at 2025-12-03: CNAME target must be ASCII or Chinese
	a.Add("SRV", rejectif.SrvHasNullTarget)        // Last verified at 2025-12-03: SRV target must not be null
	a.Add("SRV", rejectif.SrvHasEmptyTarget)       // Last verified at 2025-12-03: SRV target must not be empty
	return a.Audit(records)
}
