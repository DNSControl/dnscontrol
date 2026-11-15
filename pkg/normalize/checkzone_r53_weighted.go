package normalize

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// checkR53WeightedGroupConsistency validates that all records sharing the same
// label+type+set_identifier have identical weight and health_check_id, since
// they map to a single Route 53 ResourceRecordSet.
func checkR53WeightedGroupConsistency(records models.Records) (errs []error) {
	type groupMeta struct {
		weight      string
		healthCheck string
	}
	groups := map[string]groupMeta{}

	for _, rc := range records {
		sid := rc.Metadata["r53_set_identifier"]
		if sid == "" {
			continue
		}
		key := rc.GetLabelFQDN() + ":" + rc.Type + "!" + sid
		w := rc.Metadata["r53_weight"]
		hc := rc.Metadata["r53_health_check_id"]

		if existing, ok := groups[key]; ok {
			if existing.weight != w {
				errs = append(errs, fmt.Errorf("R53 weighted group %q at %s %s has inconsistent weights (%s vs %s)", sid, rc.Type, rc.GetLabelFQDN(), existing.weight, w))
			}
			if existing.healthCheck != hc {
				errs = append(errs, fmt.Errorf("R53 weighted group %q at %s %s has inconsistent health check IDs (%s vs %s)", sid, rc.Type, rc.GetLabelFQDN(), existing.healthCheck, hc))
			}
		} else {
			groups[key] = groupMeta{weight: w, healthCheck: hc}
		}
	}
	return errs
}
