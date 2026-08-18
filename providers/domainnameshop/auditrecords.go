package domainnameshop

import (
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	// last verified 2026-08-05
	// Domeneshop does not allow NS records at the apex: the apex nameservers
	// are managed at the delegation level and are synthesized read-only from
	// the /domains endpoint (see api.go). POSTing an apex NS record returns
	// "400 - DNS record failed validation".
	// NS records at subdomains work fine and are left untouched.
	a.Add("NS", rejectif.NsAtApex)

	return a.Audit(records)
}
