package vercel

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	// last verified 2025-11-22
	// vercel does not support custom NS records at apex (domain root)
	// vercel automatically manages apex NS records
	// attempted to set one will result in "invalid_name - Cannot set NS records at the root level. Only subdomain NS records are supported"
	a.Add("NS", rejectif.NsAtApex)

	// last verified 2025-11-22
	// bad_request - Invalid request: The specified value is not a fully qualified domain name.
	a.Add("MX", rejectif.MxNull)

	// last verified 2025-11-22
	// bad_request - Invalid request: missing required property `value`.
	a.Add("TXT", rejectif.TxtIsEmpty)

	// last verified 2025-11-22
	// bad_request - invalid_value - The specified value is not a fully qualified domain name.
	a.Add("CAA", rejectif.CaaHasEmptyTarget)

	// last verified 2025-11-22
	// Vercel misidentified extra fields in CAA record `0 issue letsencrypt.org; validationmethods=dns-01; accounturi=https://acme-v02.api.letsencrypt.org/acme/acct/1234`
	// as "cansignhttpexchanges", and add extra incorrect validation on the value
	//
	// The unit test for rejectifCaaTargetContainsUnsupportedFields is added via auditrecords_test.go
	// A vendor-specific intergration test case is added to integration_test.go
	//
	// invalid_value - Unexpected "cansignhttpexchanges" value.
	a.Add("CAA", rejectifCaaTargetContainsUnsupportedFields)

	return a.Audit(records)
}

func rejectifCaaTargetContainsUnsupportedFields(rc *models.RecordConfig) error {
	target := rc.GetTargetField()
	if !strings.Contains(target, ";") {
		return nil
	}

	parts := strings.Split(target, ";")
	// The first part is the domain, which we only check length for now
	if len(parts[0]) < 1 {
		return errors.New("caa target domain is empty")
	}
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Check if the part starts with "cansignhttpexchanges"
		// It can be just "cansignhttpexchanges" or "cansignhttpexchanges=..."
		if part == "cansignhttpexchanges" || strings.HasPrefix(part, "cansignhttpexchanges=") {
			continue
		}
		return fmt.Errorf("caa target contains unsupported field: %s", part)
	}
	return nil
}
