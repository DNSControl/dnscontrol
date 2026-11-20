package nameocean

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// maxTXTLength is the longest TXT value NameOcean stores.
const maxTXTLength = 506

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func (*nameoceanProvider) AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.Add("CAA", rejectif.CaaHasEmptyTarget) // Last verified 2026-11-20

	a.Add("CAA", caaFlagNotZeroOr128) // Last verified 2026-11-20

	a.Add("CAA", caaTagUnsupported) // Last verified 2026-11-20

	a.Add("CAA", caaTargetHasDoubleQuotes) // Last verified 2026-11-20

	a.Add("MX", rejectif.MxNull) // Last verified 2026-11-20

	a.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2026-11-20

	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2026-11-20

	a.Add("TXT", rejectif.TxtIsEmpty) // Last verified 2026-11-20

	a.Add("TXT", rejectif.TxtLongerThan(maxTXTLength)) // Last verified 2026-11-20

	errs := a.Audit(records)
	errs = append(errs, srvOnePerLabel(records)...)
	return errs
}

// caaFlagNotZeroOr128 rejects CAA flags other than 0 and 128 (critical).
func caaFlagNotZeroOr128(rc *models.RecordConfig) error {
	if f := rc.AsCAA().Flag; f != 0 && f != 128 {
		return fmt.Errorf("caa flag %d is not 0 or 128", f)
	}
	return nil
}

// caaTagUnsupported rejects CAA tags other than issue, issuewild and iodef.
func caaTagUnsupported(rc *models.RecordConfig) error {
	switch rc.AsCAA().Tag {
	case "issue", "issuewild", "iodef":
		return nil
	}
	return fmt.Errorf("caa tag %q is not issue, issuewild or iodef", rc.AsCAA().Tag)
}

// caaTargetHasDoubleQuotes rejects CAA values containing double quotes.
func caaTargetHasDoubleQuotes(rc *models.RecordConfig) error {
	if strings.Contains(rc.AsCAA().Value, `"`) {
		return errors.New("caa target has double quotes")
	}
	return nil
}

// srvOnePerLabel rejects more than one SRV record at a label: NameOcean
// stores one SRV record per name.
func srvOnePerLabel(records models.Records) []error {
	var errs []error
	seen := map[string]bool{}
	for _, rc := range records {
		if rc.Type != "SRV" {
			continue
		}
		label := rc.GetLabelFQDN()
		if seen[label] {
			errs = append(errs, fmt.Errorf("only one SRV record per label is supported (%s)", label))
			continue
		}
		seen[label] = true
	}
	return errs
}
