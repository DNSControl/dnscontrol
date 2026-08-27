package tencentdns

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
	"golang.org/x/net/idna"
)

// isValidTencentDNSString checks if a string contains only ASCII or Chinese characters.
// Tencent Cloud DNSPod allows:  a-z, A-Z, 0-9, -, and Chinese characters (汉字).
// International site doc:
// https://intl.cloud.tencent.com/zh/document/product/1295/76966
// China site doc:
// https://cloud.tencent.com/document/product/302/46277
func isValidTencentDNSString(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			// Allow CJK Unified Ideographs (Chinese characters): U+4E00 to U+9FFF
			// and CJK Extension A: U+3400 to U+4DBF
			if (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF) {
				continue
			}
			return false
		}
	}
	return true
}

// labelConstraint detects labels that contain non-ASCII characters except Chinese characters.
func labelConstraint(rc *models.RecordConfig) error {
	if !isValidTencentDNSString(rc.GetLabel()) {
		return errors.New("label contains non-ASCII characters (only Chinese is allowed)")
	}
	return nil
}

// targetConstraint detects target values that contain non-ASCII characters except Chinese characters.
// This applies to CNAME, MX, NS, SRV targets.
func targetConstraint(rc *models.RecordConfig) error {
	var target string
	switch rc.TypeNum {
	case dnsv2.TypeCNAME:
		target = rc.AsCNAME().Target
	case dnsv2.TypeMX:
		target = rc.AsMX().Mx
	case dnsv2.TypeNS:
		target = rc.AsNS().Ns
	case dnsv2.TypeSRV:
		target = rc.AsSRV().Target
	default:
		target = rc.GetRDATA().String()
	}
	if t, err := idna.ToUnicode(target); err == nil {
		target = t
	}
	if !isValidTencentDNSString(target) {
		return errors.New("target contains non-ASCII characters (only Chinese is allowed)")
	}
	return nil
}

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are
// supported, an empty list is returned.
func AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.Add("MX", rejectif.MxNull)
	a.Add("TXT", rejectif.TxtIsEmpty)
	a.Add("TXT", rejectif.TxtHasSingleQuotes)  // Tencent Cloud DNSPod alters single quotes
	a.Add("TXT", rejectif.TxtHasDoubleQuotes)  // Tencent Cloud DNSPod rejects double quotes
	a.Add("TXT", rejectif.TxtHasBackslash)     // Tencent Cloud DNSPod strips/escapes backslashes
	a.Add("TXT", rejectif.TxtHasTrailingSpace) // Tencent Cloud DNSPod strips trailing whitespace
	a.Add("SRV", rejectif.SrvHasNullTarget)
	a.Add("SRV", rejectif.SrvHasEmptyTarget)
	a.Add("*", labelConstraint)      // Tencent Cloud DNSPod only allows ASCII + Chinese, rejects other Unicode
	a.Add("CNAME", targetConstraint) // CNAME target must be ASCII or Chinese
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
