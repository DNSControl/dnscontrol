package openwrt

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

var supportedRTypes = map[string]struct{}{
	"A":     {},
	"AAAA":  {},
	"CNAME": {},
	"MX":    {},
	"SRV":   {},
}

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	// MX records cannot have null/empty target
	a.Add("MX", rejectif.MxNull)

	// SRV records cannot have null target
	a.Add("SRV", rejectif.SrvHasNullTarget)

	// Start with auditor errors
	var errors []error
	errors = append(errors, a.Audit(records)...)

	// Check for unsupported record types
	for _, rc := range records {
		if _, ok := supportedRTypes[rc.Type]; !ok {
			errors = append(errors, fmt.Errorf("record type %q is not supported by OpenWrt", rc.Type))
		}

		// OpenWrt doesn't support wildcard CNAMEs
		if rc.Type == "CNAME" && rc.GetLabel() == "*" {
			errors = append(errors, fmt.Errorf("OpenWrt does not support wildcard CNAME records"))
		}
	}

	if len(errors) == 0 {
		return nil
	}
	return errors
}
