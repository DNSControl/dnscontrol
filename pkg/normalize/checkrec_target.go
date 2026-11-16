package normalize

import (
	"errors"
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nameutil"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

// make sure target is valid reference for cnames, mx, etc.
func checkTarget(target string) error {
	if strings.ContainsAny(target, `'" +,|!£$%&()=?^*ç°§;:<>[]()@`) {
		return fmt.Errorf("target (%v) includes invalid char", target)
	}
	if !strings.HasSuffix(target, ".in-addr.arpa.") && strings.Contains(target, "/") {
		return fmt.Errorf("target (%v) includes invalid char", target)
	}
	return nil
}

// checkTargets returns zero or more errors when problems are found.
// FYI: Many of these checks are obsolete since Make*() does the same thing. We'll be removing the duplicate checks over time.
func checkTargets(rec *models.RecordConfig, domain string) (errs []error) {
	label := rec.GetLabel()
	check := func(e error) {
		if e != nil {
			err := fmt.Errorf("%s: %s %s: %s", rec.FilePos, rec.Type, rec.GetLabelFQDN(), e.Error())
			if _, ok := e.(Warning); ok {
				err = Warning{err}
			}
			errs = append(errs, err)
		}
	}
	switch rec.Type { // #rtype_variations

	// IMPORT_TRANSFORM is not in the record catalog, so the default case would reject it.
	case "IMPORT_TRANSFORM":

	case "ALIAS":
		check(checkTarget(rec.AsALIAS().Target))
	case "CNAME":
		check(checkTarget(rec.AsCNAME().Target))
		if label == "@" {
			check(errors.New("cannot create CNAME record for bare domain. Use ALIAS"))
		}
		labelFQDN := nameutil.ToFqdnNoDot(label, domain)
		targetFQDN := nameutil.ToFqdnNoDot(rec.AsCNAME().Target, domain)
		if labelFQDN == targetFQDN {
			check(errors.New("CNAME loop (target points at itself)"))
		}
	case "DNAME":
		check(checkTarget(rec.AsDNAME().Target))
	case "MX":
		check(checkTarget(rec.AsMX().Mx))
	case "NAPTR":
		target := rec.AsNAPTR().Replacement
		if target != "" {
			check(checkTarget(target))
		}
	case "NS":
		check(checkTarget(rec.AsNS().Ns))
		if label == "@" {
			check(errors.New("cannot create NS record for bare domain. Use NAMESERVER instead"))
		}
	case "PTR":
		check(checkTarget(rec.AsPTR().Ptr))
	case "SOA":
		f := rec.AsSOA()
		check(checkSoa(f.Expire, f.Minttl, f.Refresh, f.Retry, f.Mbox))
		check(checkTarget(f.Ns))
		if label != "@" {
			check(errors.New("SOA record is only valid for bare domain"))
		}
	case "SRV":
		check(checkTarget(rec.AsSRV().Target))
	case "LUA":
		f := rec.AsLUA()
		upper := strings.ToUpper(f.LuaType)
		if upper == "" {
			check(errors.New("LUA records must specify an emitted rtype"))
			break
		}
		if _, ok := dnsv2.StringToType[upper]; !ok {
			check(fmt.Errorf("LUA emitted rtype (%s) is not a valid DNS type", f.LuaType))
		}
	default:
		if _, known := privatetypes.LookupRecordType(rec.Type); known && rec.GetRDATA() != nil {
			// Catalog records have already been parsed and validated. Provider
			// support is checked separately.
			return errs
		}
		errs = append(errs, fmt.Errorf("checkTargets: Unimplemented record type (%v) domain=%v name=%v",
			rec.Type, domain, rec.GetLabel()))
	}
	return errs
}
