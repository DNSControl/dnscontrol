package models

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"

	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
	dnsv1 "github.com/miekg/dns"
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

// GetTargetCombinedFunc returns all the rdata fields of a RecordConfig as one
// string. How TXT records are encoded is defined by encodeFn.  If encodeFn is
// nil the TXT data is returned unaltered.
func (rc *RecordConfig) GetTargetCombinedFunc(encodeFn func(s string) string) string {
	if rc.Type == "TXT" || rc.Type == "LUA" {
		if encodeFn == nil {
			return rc.GetTargetField()
		}
		return encodeFn(rc.GetTargetField())
	}
	return rc.GetTargetCombined()
}

// GetTargetCombined returns a string with the various fields combined.
// For example, an MX record might output `10 mx10.example.tld`.
func (rc *RecordConfig) GetTargetCombined() string {
	// TXT presentation must split the value into quoted, 255-octet
	// character-strings (this is the form providers send to their APIs, e.g.
	// PowerDNS). The stored rdata is the raw text (the single source of truth),
	// so quote/chunk it here.
	if rc.Type == "TXT" {
		return txtutil.EncodeQuoted(rc.GetTargetTXTJoined())
	}
	if rc.GetRDATA() != nil {
		return rc.GetRDATA().String()
	}

	// Pseudo records:
	if _, ok := dnsv1.StringToType[rc.Type]; !ok {
		switch rc.Type { // #rtype_variations
		case "LUA":
			return rc.luaCombined()
		case "R53_ALIAS":
			return rc.GetRDATA().String()
		case "AZURE_ALIAS":
			// Differentiate between multiple AZURE_ALIASs on the same label.
			return fmt.Sprintf("%s atype=%s", rc.target, rc.AzureAlias["type"])
		case "AKAMAITLC":
			return fmt.Sprintf("%s %s", rc.AnswerType, rc.target)
		default:
			// Just return the target.
			return rc.target
		}
	}

	// Everything else
	switch rc.Type {
	case "UNKNOWN":
		return fmt.Sprintf("rtype=%s rdata=%s", rc.UnknownTypeName, rc.target)
	case "TXT":
		return rc.zoneFileQuoted()
	case "SOA":
		panic("SOA converted")
		// return fmt.Sprintf("%s %v %d %d %d %d %d", rc.target, rc.SoaMbox, rc.SoaSerial, rc.SoaRefresh, rc.SoaRetry, rc.SoaExpire, rc.SoaMinttl)
	}

	return rc.zoneFileQuoted()
}

// zoneFileQuoted returns the rData as would be quoted in a zonefile.
func (rc *RecordConfig) zoneFileQuoted() string {
	// We cheat by converting to a dns.RR and use the String() function.
	// This combines all the data for us, and even does proper quoting.
	// Sadly String() always includes a header, which we must strip out.
	// TODO(tlim): Request the dns project add a function that returns
	// the string without the header.
	if rc.Type == "NAPTR" && rc.GetTargetField() == "" {
		rc.MustSetTarget(".")
	}

	if rc.GetRDATA() != nil {
		return rc.GetRDATA().String()
	}

	rr := rc.ToRR()
	header := rr.Header().String()
	full := rr.String()
	if !strings.HasPrefix(full, header) {
		panic("assertion failed. dns.Hdr.String() behavior has changed in an incompatible way")
	}
	return full[len(header):]
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

// GetTargetRFC1035Quoted returns the target as it would be in an
// RFC1035-style zonefile.
// Do not use this function if RecordConfig might be a pseudo-rtype
// such as R53_ALIAS.  Use GetTargetCombined() instead.
func (rc *RecordConfig) GetTargetRFC1035Quoted() string {
	return rc.zoneFileQuoted()
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
		return fmt.Sprintf("%q", rc.GetTargetCombined())
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
