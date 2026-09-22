package mittwald

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// Limits of the mStudio API (OpenAPI schema de.mittwald.v1.dns.*).
const (
	maxMXPrio    = 100
	maxTXTLength = 2048
)

// maxPerSet is the number of records a record set of one name can hold.
var maxPerSet = map[uint16]int{
	dnsv2.TypeA:    10,
	dnsv2.TypeAAAA: 10,
	dnsv2.TypeMX:   10,
	dnsv2.TypeTXT:  20,
}

// labelRE is what the API accepts for each label of a name.
var labelRE = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_-]{0,62}$`)

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.TypesSupported([]string{"A", "AAAA", "CAA", "CNAME", "MX", "SRV", "TXT"})
	a.Add("*", rejectif.LabelIsWildcard) // Last verified 2026-09-22
	a.Add("*", labelHasInvalidChars)     // Last verified 2026-09-22
	a.Add("CAA", caaTagUnsupported)
	a.Add("CAA", caaValueNotHostname) // Last verified 2026-09-22
	a.Add("MX", rejectif.MxNull)      // Last verified 2026-09-22
	a.Add("MX", mxPreferenceTooHigh)  // Last verified 2026-09-22
	a.Add("SRV", rejectif.SrvHasNullTarget)
	a.Add("TXT", rejectif.TxtHasTrailingSpace) // Last verified 2026-09-22
	a.Add("TXT", rejectif.TxtIsEmpty)          // Last verified 2026-09-22
	a.Add("TXT", rejectif.TxtLongerThan(maxTXTLength))

	return append(a.Audit(records), tooManyPerSet(records)...)
}

func labelHasInvalidChars(rc *models.RecordConfig) error {
	if rc.Name == "@" {
		return nil
	}
	for l := range strings.SplitSeq(rc.Name, ".") {
		if !labelRE.MatchString(l) {
			return fmt.Errorf("label %q is not accepted by mStudio", l)
		}
	}
	return nil
}

func caaTagUnsupported(rc *models.RecordConfig) error {
	switch rc.AsCAA().Tag {
	case "issue", "issuewild", "iodef":
		return nil
	}
	return errors.New("CAA tag must be issue, issuewild or iodef")
}

// hostnameRE is a host name; the API takes nothing else as a CAA value, so
// neither parameters ("ca.example; account=1") nor an iodef URL.
var hostnameRE = regexp.MustCompile(`^[a-zA-Z0-9_]([a-zA-Z0-9_-]{0,62})(\.[a-zA-Z0-9_][a-zA-Z0-9_-]{0,62})*\.?$`)

func caaValueNotHostname(rc *models.RecordConfig) error {
	if !hostnameRE.MatchString(rc.AsCAA().Value) {
		return errors.New("mStudio takes a host name as CAA value, without parameters")
	}
	return nil
}

func mxPreferenceTooHigh(rc *models.RecordConfig) error {
	if rc.AsMX().Preference > maxMXPrio {
		return fmt.Errorf("MX preference above %d", maxMXPrio)
	}
	return nil
}

// tooManyPerSet reports names with more records of one type than a record set can hold.
func tooManyPerSet(records models.Records) []error {
	type key struct {
		name    string
		typeNum uint16
	}
	counts := map[key]int{}
	var errs []error
	for _, rc := range records {
		k := key{rc.NameFQDN, rc.TypeNum}
		counts[k]++
		if limit, ok := maxPerSet[rc.TypeNum]; ok && counts[k] == limit+1 {
			errs = append(errs, fmt.Errorf("%s: more than %d %s records", rc.NameFQDN, limit, rc.Type))
		}
	}
	return errs
}
