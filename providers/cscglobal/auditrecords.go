package cscglobal

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull) // Last verified 2022-06-07

	a.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2020-12-28

	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2022-08-08

	a.Add("TXT", rejectif.TxtHasTrailingSpace) // Last verified 2022-06-10

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2023-12-03

	return a.Audit(records)
}

/* How To Write Providers:

Each test should be encapsulated in a function that can be tested
individually.  If the test is of general use, add it to the
rejectif module.

The "Last verified" comment logs the last time we verified this
test was needed.  Sometimes companies change their API.  Once a year,
try removing tests one at a time to verify they are still needed.

*/
