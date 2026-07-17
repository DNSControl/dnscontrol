package tencentdns

import (
	"fmt"
	"strconv"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull)
	a.Add("TXT", rejectif.TxtIsEmpty)
	a.Add("SRV", rejectif.SrvHasNullTarget)
	a.Add("SRV", rejectif.SrvHasEmptyTarget)
	a.Add("*", rejectifInvalidRecordWeight)

	return a.Audit(records)
}

func rejectifInvalidRecordWeight(rc *models.RecordConfig) error {
	weight := rc.Metadata[metaRecordWeight]
	if weight == "" {
		return nil
	}

	parsed, err := strconv.ParseUint(weight, 10, 64)
	if err != nil {
		return fmt.Errorf("%s %q is not a valid integer on %s %s", metaRecordWeight, weight, rc.Type, rc.GetLabelFQDN())
	}
	if parsed > 100 {
		return fmt.Errorf("%s %d must be between 0 and 100 on %s %s", metaRecordWeight, parsed, rc.Type, rc.GetLabelFQDN())
	}
	return nil
}
