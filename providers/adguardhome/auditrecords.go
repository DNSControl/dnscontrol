package adguardhome

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

var supportedRTypes = map[string]struct{}{
	"A":                            {},
	"AAAA":                         {},
	"CNAME":                        {},
	"ALIAS":                        {},
	"ADGUARDHOME_A_PASSTHROUGH":    {},
	"ADGUARDHOME_AAAA_PASSTHROUGH": {},
}

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("ALIAS", rejectif.LabelNotApex)

	a.Add("ADGUARDHOME_A_PASSTHROUGH", nonNullValue)

	a.Add("ADGUARDHOME_AAAA_PASSTHROUGH", nonNullValue)

	errors := []error{}
	errors = append(errors, a.Audit(records)...)

	for _, r := range records {
		if _, ok := supportedRTypes[r.Type]; !ok {
			errors = append(errors, fmt.Errorf("%s rtype is not supported", r.Type))
		}
	}

	return errors
}

func nonNullValue(v *models.RecordConfig) error {
	if len(v.GetTargetField()) != 0 {
		return fmt.Errorf("%s rtype value should be empty", v.Type)
	}

	return nil
}
