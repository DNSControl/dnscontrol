package models

import (
	"fmt"
	"net/netip"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

/* .target is kind of a mess.
If an rType has more than one field, one field goes in .target and the remaining are stored in bespoke fields.
Not the best design, but we're stuck with it until we re-do RecordConfig, possibly using generics.
*/

// GetTargetField returns the target. There may be other fields, but they are
// not included. For example, the Preference field of an MX record isn't included.
func (rc *RecordConfig) GetTargetField() string {
	if rc.Type == "TXT" {
		// TXT stores its value in .rdata (the single source of truth); .target
		// is not populated for TXT.
		return rc.GetTargetTXTJoined()
	}
	if rc.rdata != nil {
		if rc.Type == "R53_ALIAS" {
			// R53_ALIAS's target (DNSName) is not the last field of the RDATA
			// (that's the zone_id), so the "last field" heuristic below is wrong
			// for it.
			return rc.AsR53ALIAS().Target
		}
		// Return the last field. Not perfect, but good enough until we get rid of this function.
		fx, err := RDtoFieldsStrings(rc.GetRDATA())
		if err != nil {
			return rc.GetRDATA().String()
		}
		if len(fx) == 0 {
			return ""
		}
		return fx[len(fx)-1]
	}
	// return rc.target
	panic("GetTargetField")
}

// GetTargetIP returns the net.IP stored in .target.
func (rc *RecordConfig) GetTargetIP() netip.Addr {
	switch f := rc.GetRDATA().(type) {
	case dnsrdatav2.A:
		return f.Addr
	case dnsrdatav2.AAAA:
		return f.Addr
	}
	panic(fmt.Sprintf("wrong type GetTargetIP(%T)", rc.GetRDATA()))
	// ip, _ := netip.ParseAddr(rc.target)
	// return ip
}

// GetTargetDebug returns a string with the various fields spelled out.
func (rc *RecordConfig) GetTargetDebug() string {
	var content strings.Builder
	fmt.Fprintf(&content, "%s %s %d %q", rc.Type, rc.NameFQDN, rc.TTL, rc.GetRDATA().String())
	for k, v := range rc.Metadata {
		fmt.Fprintf(&content, " %s=%q", k, v)
	}
	return content.String()
}

//// GetTargetJS returns the target as a JavaScript literal, as documented in
//// documentation/language-reference/domain-modifiers/*.md. Each parameter is
//// quoted, unless it is an integer or boolean.  We can't use .String()
//// because it is not designed for JavaScript and may include unquoted
//// parameters, which would break the JavaScript.  Instead, we must quote each
//// parameter separately. This doesn't support all types and needs to be improved.
//// FIXME(tlim): This duplicates code in commands/getZones.go:formatDsl().
////
////	We should extract the common logic into a function they can both use.
//func (rc *RecordConfig) GetTargetJS() string {
//	//	if rc.Type == "TXT" {
//	//		encoded, err := json.Marshal(rc.GetTargetTXTSegmented())
//	//		if err != nil {
//	//			panic(err) // strings are always JSON-marshalable
//	//		}
//	//		return string(encoded)
//	//	}
//	if rc.Type == "LUA" {
//		return fmt.Sprintf("%q", rc.GetTargetField())
//	}
//	switch rc.Type {
//	// case "A", "AAAA", "AKAMAICDN", "CNAME", "DHCID", "NS", "OPENPGPKEY", "PTR":
//	//
//	//	return fmt.Sprintf("%q", rc.target)
//	//
//	// case "SOA":
//	//
//	//	// SOA(ns, mbox, refresh, retry, expire, minttl)
//	//	f := rc.AsSOA()
//	//	return fmt.Sprintf("%q, %q, %d, %d, %d, %d", f.Ns, f.Mbox, f.Refresh, f.Retry, f.Expire, f.Minttl)
//	//
//	// case "SRV":
//	//
//	//	// SRV(priority, weight, port, target)
//	//	f := rc.AsSRV()
//	//	return fmt.Sprintf("%d, %d, %d, %q", f.Priority, f.Weight, f.Port, f.Target)
//	//
//	// default:
//	//
//	//		return fmt.Sprintf("%q", rc.GetRDATA().String())
//	//	}
//	}
//	s, _ := RDtoFieldsJS(rc.GetRDATA())
//	return strings.Join(s, ", ")
//}

// SetTarget sets the target, assuming that the rtype is appropriate.
// func (rc *RecordConfig) SetTarget(target string) error {
// 	// TXT stores its value in .rdata (the single source of truth). Route legacy
// 	// SetTarget callers there so .target is never the TXT store.
// 	if rc.Type == "TXT" {
// 		return rc.SetTargetTXT(target)
// 	}
// 	rc.target = target
// 	return nil
// }

// // MustSetTarget is like SetTarget, but panics if an error occurs.
// // It should only be used in _test.go files and in the init() function.
// func (rc *RecordConfig) MustSetTarget(target string) {
// 	if err := rc.SetTarget(target); err != nil {
// 		panic(err)
// 	}
// }

// SetTargetIP sets the target to an IP, verifying this is an appropriate rtype.
func (rc *RecordConfig) SetTargetIP(ip netip.Addr) error {
	// TODO(tlim): Verify the rtype is appropriate for an IP.
	//return rc.SetTarget(ip.String())
	switch rc.TypeNum {
	case dnsv2.TypeA:
		rd, err := MakeA("", nil, nrc.Flags{}, ip)
		if err != nil {
			return err
		}
		rc.SetRDATA(rd)
		return nil
	case dnsv2.TypeAAAA:
		rd, err := MakeAAAA("", nil, nrc.Flags{}, ip)
		if err != nil {
			return err
		}
		rc.SetRDATA(rd)
		return nil
	}
	return fmt.Errorf("invalid IP %v", ip)
}
