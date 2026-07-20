package unifi

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// Supported record types for UniFi Network
// Note: NS records are supported but only with IP addresses, not hostnames.
var supportedRTypes = map[string]struct{}{
	"A":     {},
	"AAAA":  {},
	"CNAME": {},
	"MX":    {},
	"TXT":   {},
	"SRV":   {},
}

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	// TXT records have limitations
	a.Add("TXT", rejectif.TxtIsEmpty)
	a.Add("TXT", rejectif.TxtLongerThan(255)) // UniFi limits TXT to 255 chars per record

	// MX records cannot have null/empty target
	a.Add("MX", rejectif.MxNull)

	// SRV records cannot have null target
	a.Add("SRV", rejectif.SrvHasNullTarget)

	// Start with auditor errors
	var errors []error
	errors = append(errors, a.Audit(records)...)

	// Check for unsupported record types
	for _, r := range records {
		if _, ok := supportedRTypes[r.Type]; !ok {
			errors = append(errors, fmt.Errorf("record type %q is not supported by UniFi", r.Type))
		}

		// UniFi doesn't support wildcard CNAMEs well
		if r.Type == "CNAME" && r.GetLabel() == "*" {
			errors = append(errors, fmt.Errorf("UniFi does not support wildcard CNAME records"))
		}
	}

	if len(errors) == 0 {
		return nil
	}
	return errors
}
