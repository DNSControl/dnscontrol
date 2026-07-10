package huaweicloud

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}
	// go test -v -run 'TestDNSProviders/.*/.*NullMX(Apex)?:(create|unnull|renull)$' -args -verbose -profile HUAWEICLOUD
	a.Add("MX", rejectif.MxNull) // Last verified 2026-05-18
	// go test -v -run 'TestDNSProviders/.*/.*TXT backslashes:TXT with backslashs$' -args -verbose -profile HUAWEICLOUD
	a.Add("TXT", rejectif.TxtHasBackslash) // Last verified 2026-05-18
	// go test -v -run 'TestDNSProviders/.*/.*complex TXT:TXT with (1 dq-1interior|2 dq-2interior|1 dq-left|1 dq-right)$' -args -verbose -profile HUAWEICLOUD
	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2026-05-18

	return a.Audit(records)
}
