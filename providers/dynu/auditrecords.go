package dynu

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.TypesSupported([]string{
		"A", "AAAA", "AFSDB", "CAA", "CERT", "CNAME", "DHCID", "DNAME",
		"HINFO", "HTTPS", "KEY", "LOC", "MX", "NAPTR", "NS", "OPENPGPKEY",
		"PTR", "RP", "SMIMEA", "SRV", "SSHFP", "SVCB", "TLSA", "TXT", "URI",
	}) // Last verified 2026-06-16

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2026-06-16

	a.Add("*", rejectif.LabelIsWildcard) // Last verified 2026-06-16

	return a.Audit(records)
}
