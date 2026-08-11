package hetznerv2

import (
	"github.com/DNSControl/dnscontrol/v4/models"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(_ []*models.RecordConfig) []error {
	return nil
}
