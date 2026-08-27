package sakuracloud

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull) // Last verified 2024-08-09

	a.Add("SRV", rejectif.SrvHasNullTarget) // Last verified 2024-08-09

	a.Add("TXT", rejectif.TxtHasBackslash) // Last verified 2024-08-09

	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2024-08-09

	a.Add("TXT", rejectif.TxtHasUnpairedDoubleQuotes) // Last verified 2024-08-09

	a.Add("TXT", rejectif.TxtLongerThan(500)) // Last verified 2024-08-09

	a.Add("CAA", rejectifCaaLongerThan(64)) // Last verified 2024-08-09

	a.Add("NS", rejectifNsPointsToOrigin) // Last verified 2024-08-09

	for _, t := range []string{"ALIAS", "CNAME", "HTTPS", "MX", "NS", "PTR", "SRV", "SVCB"} {
		a.Add(t, rejectifTargetHasExample) // Last verified 2024-08-09
	}

	for _, t := range []string{"A", "AAAA", "ALIAS", "CAA", "CNAME", "HTTPS", "MX", "NS", "PTR", "SRV", "SVCB", "TXT"} {
		a.Add(t, rejectifLabelHasExample) // Last verified 2024-08-09
	}
	return a.Audit(records)
}

// rejectifCaaLongerThan returns a function that audits CAA records
// where the length of the property value is greater than maxLength.
func rejectifCaaLongerThan(maxLength int) func(rc *models.RecordConfig) error {
	return func(rc *models.RecordConfig) error {
		m := maxLength
		if len(rc.AsCAA().Value) > m {
			return fmt.Errorf("CAA record longer than %d octets (chars)", m)
		}
		return nil
	}
}

// rejectifNsPointsToOrigin audits NS records that point to the origin.
func rejectifNsPointsToOrigin(rc *models.RecordConfig) error {
	originFQDN := strings.TrimPrefix(rc.GetLabelFQDN(), rc.GetLabel()+".") + "."
	if originFQDN == rc.AsNS().Ns {
		return fmt.Errorf("NS record points to the origin: %s", rc.AsNS().Ns)
	}
	return nil
}

var labelExampleRe = regexp.MustCompile(`^example[0-9]?$`)

func hasLabelExample(domain string) error {
	labels := strings.Split(strings.TrimSuffix(domain, "."), ".")
	if slices.ContainsFunc(labels, labelExampleRe.MatchString) {
		return fmt.Errorf("label contains `example`: %s", domain)
	}
	return nil
}

// rejectifTargetHasExample returns a function that audits RDATA targets
// containing the following labels:
//
//   - example
//   - exampleN, where N is a numerical character
func rejectifTargetHasExample(rc *models.RecordConfig) error {
	var target string
	switch rc.TypeNum {
	case privatetypes.TypeALIAS:
		target = rc.AsALIAS().Target
	case dnsv2.TypeCNAME:
		target = rc.AsCNAME().Target
	case dnsv2.TypeHTTPS:
		target = rc.AsHTTPS().Target
	case dnsv2.TypeMX:
		target = rc.AsMX().Mx
	case dnsv2.TypeNS:
		target = rc.AsNS().Ns
	case dnsv2.TypePTR:
		target = rc.AsPTR().Ptr
	case dnsv2.TypeSRV:
		target = rc.AsSRV().Target
	case dnsv2.TypeSVCB:
		target = rc.AsSVCB().Target
	}
	return hasLabelExample(target)
}

// rejectifLabelHasExample returns a function that audits owner names
// containing the following labels:
//
//   - example
//   - exampleN, where N is a numerical character
func rejectifLabelHasExample(rc *models.RecordConfig) error {
	return hasLabelExample(rc.GetLabel())
}
