package models

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"

	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

/* .target is kind of a mess.
If an rType has more than one field, one field goes in .target and the remaining are stored in bespoke fields.
Not the best design, but we're stuck with it until we re-do RecordConfig, possibly using generics.
*/

// GetTargetField returns the target. There may be other fields, but they are
// not included. For example, the .MxPreference field of an MX record isn't included.
func (rc *RecordConfig) GetTargetField() string {
	if rc.Type == "TXT" {
		// TXT stores its value in .rdata (the single source of truth); .target
		// is not populated for TXT.
		return rc.GetTargetTXTJoined()
	}
	if rc.rdata != nil && (rc.Type == "A" || rc.Type == "AAAA" || rc.Type == "CNAME" || rc.Type == "NX") {
		return rc.GetRDATA().String()
	}
	return rc.target
}

// GetTargetIP returns the net.IP stored in .target.
func (rc *RecordConfig) GetTargetIP() netip.Addr {
	if rc.Type != "A" && rc.Type != "AAAA" {
		panic(fmt.Errorf("GetTargetIP called on an inappropriate rtype (%s)", rc.Type))
	}

	if rc.GetRDATA() != nil {
		if rc.Type == "A" {
			return rc.rdata.(dnsrdatav2.A).Addr
		}
		if rc.Type == "AAAA" {
			return rc.rdata.(dnsrdatav2.AAAA).Addr
		}
	}

	ip, _ := netip.ParseAddr(rc.target)
	return ip
}

func (rc *RecordConfig) luaCombined() string {
	rtype := rc.luaTypeUpper()
	payload := rc.target
	if rtype == "" {
		return payload
	}
	payload = txtutil.EncodeQuoted(payload)
	if payload == "" {
		return rtype
	}
	return fmt.Sprintf("%s %s", rtype, payload)
}

func (rc *RecordConfig) luaTypeUpper() string {
	if rc.LuaRType == "" {
		return ""
	}
	return strings.ToUpper(rc.LuaRType)
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

// GetTargetJS returns the target as a JavaScript literal, as documented in
// documentation/language-reference/domain-modifiers/*.md. Each parameter is
// quoted, unless it is an integer or boolean.  We can't use GetTargetCombined()
// because it is not designed for JavaScript and may include unquoted
// parameters, which would break the JavaScript.  Instead, we must quote each
// parameter separately. This doesn't support all types and needs to be improved.
// FIXME(tlim): This duplicates code in commands/getZones.go:formatDsl().
//
//	We should extract the common logic into a function they can both use.
func (rc *RecordConfig) GetTargetJS() string {
	if rc.Type == "TXT" {
		encoded, err := json.Marshal(rc.GetTargetTXTSegmented())
		if err != nil {
			panic(err) // strings are always JSON-marshalable
		}
		return string(encoded)
	}
	if rc.Type == "LUA" {
		return fmt.Sprintf("%q", rc.GetTargetField())
	}
	switch rc.Type {
	case "A", "AAAA", "AKAMAICDN", "CNAME", "DHCID", "NS", "OPENPGPKEY", "PTR":
		return fmt.Sprintf("%q", rc.target)
	case "SOA":
		// SOA(ns, mbox, refresh, retry, expire, minttl)
		f := rc.AsSOA()
		return fmt.Sprintf("%q, %q, %d, %d, %d, %d", f.Ns, f.Mbox, f.Refresh, f.Retry, f.Expire, f.Minttl)
	case "SRV":
		// SRV(priority, weight, port, target)
		return fmt.Sprintf("%d, %d, %d, %q", rc.SrvPriority, rc.SrvWeight, rc.SrvPort, rc.target)
	default:
		return fmt.Sprintf("%q", rc.GetRDATA().String())
	}
}

// SetTarget sets the target, assuming that the rtype is appropriate.
func (rc *RecordConfig) SetTarget(target string) error {
	// TXT stores its value in .rdata (the single source of truth). Route legacy
	// SetTarget callers there so .target is never the TXT store.
	if rc.Type == "TXT" {
		return rc.SetTargetTXT(target)
	}
	rc.target = target
	return nil
}

// MustSetTarget is like SetTarget, but panics if an error occurs.
// It should only be used in _test.go files and in the init() function.
func (rc *RecordConfig) MustSetTarget(target string) {
	if err := rc.SetTarget(target); err != nil {
		panic(err)
	}
}

// SetTargetIP sets the target to an IP, verifying this is an appropriate rtype.
func (rc *RecordConfig) SetTargetIP(ip netip.Addr) error {
	// TODO(tlim): Verify the rtype is appropriate for an IP.
	return rc.SetTarget(ip.String())
}
