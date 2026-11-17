package openwrt

import (
	"errors"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func (*openwrtProvider) AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	// MX records cannot have null/empty target
	a.Add("MX", rejectif.MxNull)

	// SRV records cannot have null target
	a.Add("SRV", rejectif.SrvHasNullTarget)

	// Start with auditor errors
	var errs []error
	errs = append(errs, a.Audit(records)...)

	for _, rc := range records {
		// OpenWrt doesn't support wildcard CNAMEs
		if rc.Type == "CNAME" && rc.GetLabel() == "*" {
			errs = append(errs, errors.New("OpenWrt does not support wildcard CNAME records"))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}
