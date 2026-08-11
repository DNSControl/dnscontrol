package route53

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	a.Add("R53_ALIAS", rejectifTargetEqualsLabel) // Last verified 2023-03-01
	a.Add("*", rejectifInvalidR53Weight)

	return a.Audit(records)
}

// Normally this kind of function would be put in `pkg/rejectif` but
// since this is ROUTE53-specific, we'll include it here.

// rejectifTargetEqualsLabel rejects an ALIAS that would create a loop.
func rejectifTargetEqualsLabel(rc *models.RecordConfig) error {
	if (rc.GetLabelFQDN() + ".") == rc.GetTargetField() {
		return errors.New("alias target loop")
	}
	return nil
}

// rejectifInvalidR53Weight validates Route 53 weighted routing metadata.
func rejectifInvalidR53Weight(rc *models.RecordConfig) error {
	weight := rc.Metadata["r53_weight"]
	setID := rc.Metadata["r53_set_identifier"]

	if weight == "" && setID == "" {
		return nil
	}

	if weight != "" && setID == "" {
		return fmt.Errorf("r53_weight is set but r53_set_identifier is missing on %s %s", rc.Type, rc.GetLabelFQDN())
	}
	if weight == "" && setID != "" {
		return fmt.Errorf("r53_set_identifier is set but r53_weight is missing on %s %s", rc.Type, rc.GetLabelFQDN())
	}

	w, err := strconv.ParseInt(weight, 10, 64)
	if err != nil {
		return fmt.Errorf("r53_weight %q is not a valid integer on %s %s", weight, rc.Type, rc.GetLabelFQDN())
	}
	if w < 0 || w > 255 {
		return fmt.Errorf("r53_weight %d must be between 0 and 255 on %s %s", w, rc.Type, rc.GetLabelFQDN())
	}

	if len(setID) > 128 {
		return fmt.Errorf("r53_set_identifier must be 128 characters or fewer on %s %s", rc.Type, rc.GetLabelFQDN())
	}

	return nil
}
