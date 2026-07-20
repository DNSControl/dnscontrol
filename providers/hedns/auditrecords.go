package hedns

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// dynamicAllowedTypes lists the record types that support Dynamic DNS on HE DNS.
var dynamicAllowedTypes = map[string]bool{
	"A":    true,
	"AAAA": true,
	"TXT":  true,
}

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull) // Last verified 2020-12-28

	// hedns_dynamic and hedns_ddns_key are only valid for A, AAAA, TXT.
	a.Add("*", rejectDynamicOnUnsupportedType)

	return a.Audit(records)
}

func rejectDynamicOnUnsupportedType(rc *models.RecordConfig) error {
	if dynamicAllowedTypes[rc.Type] {
		return nil
	}
	if rc.Metadata == nil {
		return nil
	}
	if v := rc.Metadata[metaDynamic]; v != "" && v != "off" {
		return fmt.Errorf("%s record %q: %s is only supported on A, AAAA, and TXT records", rc.Type, rc.GetLabel(), metaDynamic)
	}
	if rc.Metadata[metaDDNSKey] != "" {
		return fmt.Errorf("%s record %q: %s is only supported on A, AAAA, and TXT records", rc.Type, rc.GetLabel(), metaDDNSKey)
	}
	return nil
}
