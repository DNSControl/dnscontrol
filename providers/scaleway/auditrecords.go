package scaleway

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
	a.Add("TXT", rejectif.TxtIsEmpty)

	// The API trims a TXT value before storing it, so a leading or trailing
	// space is dropped without being reported. Left alone that reads as a
	// permanently pending change: the record never matches what was asked for.
	// Interior whitespace is kept. Last verified 2026-09-19.
	a.Add("TXT", rejectif.TxtStartsOrEndsWithSpaces)

	// Last verified 2026-09-19.
	a.Add("TXT", txtStartsOrEndsWithDoubleQuote)

	return a.Audit(records)
}

// txtStartsOrEndsWithDoubleQuote rejects TXT values that begin or end with a
// double quote. Scaleway drops a leading or trailing double quote server-side,
// so such a value can never round-trip. Interior double quotes are stored
// as-is and are therefore not rejected.
func txtStartsOrEndsWithDoubleQuote(rc *models.RecordConfig) error {
	txt := rc.GetTargetTXTJoined()
	if txt == "" {
		return nil
	}
	if txt[0] == '"' || txt[len(txt)-1] == '"' {
		return errors.New("txtstring starts or ends with a doublequote")
	}
	return nil
}
