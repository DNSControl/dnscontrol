package models

import (
	"fmt"
	"net/netip"
)

// PopulateFromStringFunc populates a RecordConfig by parsing a common RFC1035-like format.
//
//	rtype: the resource record type (rtype)
//	contents: a string that contains all parameters of the record's rdata (see below)
//	txtFn: If rtype == "TXT", this function is used to parse contents, or nil if no parsing is needed.
//
// The "contents" field is the format used in RFC1035 zonefiles. It is the text
// after the rtype.  For example, in the line: foo IN MX 10 mx.example.com.
// contents stores everything after the "MX" (not including the space).
//
// Typical values for txtFn include:
//
//	nil:  no parsing required.
//	txtutil.ParseQuoted: Parse via Tom's interpretation of RFC1035.
//	txtutil.ParseCombined: Backwards compatible with Parse via miekg's interpretation of RFC1035.
//
// Many providers deliver record data in this format or something close to it.
// This function is provided to reduce the amount of duplicate code across
// providers.  If a particular rtype is not handled as a particular provider
// expects, simply handle it beforehand as a special case.
//
// Example 1: Normal use.
//
//	rtype := FILL_IN_RTYPE
//	rc := &models.RecordConfig{Type: rtype, TTL: FILL_IN_TTL}
//	rc.SetLabelFromFQDN(FILL_IN_NAME, origin)
//	rc.Original = FILL_IN_ORIGINAL // The raw data received from provider (if needed later)
//	if err = rc.PopulateFromStringFunc(rtype, target, origin, nil); err != nil {
//		return nil, fmt.Errorf("unparsable record type=%q received from PROVDER_NAME: %w", rtype, err)
//	}
//	return rc, nil
//
// Example 2: Use your own MX parser.
//
//	rtype := FILL_IN_RTYPE
//	rc := &models.RecordConfig{Type: rtype, TTL: FILL_IN_TTL}
//	rc.SetLabelFromFQDN(FILL_IN_NAME, origin)
//	rc.Original = FILL_IN_ORIGINAL // The raw data received from provider (if needed later)
//	switch rtype {
//	case "MX":
//		// MX priority in a separate field.
//		err = rc.SetTargetMX(cr.Priority, target)
//	default:
//		err = rc.PopulateFromString(rtype, target, origin)
//	}
//	if err != nil {
//		return nil, fmt.Errorf("unparsable record type=%q received from PROVDER_NAME: %w", rtype, err)
//	}
//	return rc, nil
func (rc *RecordConfig) PopulateFromStringFunc(rtype, contents, origin string, txtFn func(s string) (string, error)) error {
	if rc.Type != "" && rc.Type != rtype {
		return fmt.Errorf("assertion failed: rtype already set (%s) (%s)", rtype, rc.Type)
	}

	switch rc.Type = rtype; rtype { // #rtype_variations
	case "A":
		ip, err := netip.ParseAddr(contents)
		if err != nil || !ip.Is4() {
			return fmt.Errorf("invalid IP in A record: %s", contents)
		}
		return rc.SetTargetIP(ip) // Reformat to canonical form.
	case "AAAA":
		ip, err := netip.ParseAddr(contents)
		if err != nil || !ip.Is6() {
			return fmt.Errorf("invalid IP in AAAA record: %s", contents)
		}
		return rc.SetTargetIP(ip) // Reformat to canonical form.
	case "AKAMAICDN", "AKAMAITLC", "ALIAS", "ANAME", "CNAME", "NS", "PTR":
		return rc.SetTarget(contents)
	case "CAA":
		return rc.SetTargetCAAString(contents)
	case "DS":
		return rc.SetTargetDSString(contents)
	case "DNSKEY":
		return rc.SetTargetDNSKEYString(contents)
	case "DHCID":
		return rc.SetTarget(contents)
	case "DNAME":
		return rc.SetTarget(contents)
	case "LOC":
		return rc.SetTargetLOCString(origin, contents)
	case "MX":
		return rc.SetTargetMXString(contents)
	case "NAPTR":
		return rc.SetTargetNAPTRString(contents)
	case "OPENPGPKEY":
		return rc.SetTarget(contents)
	case "SMIMEA":
		return rc.SetTargetSMIMEAString(contents)
	case "SOA":
		return rc.SetTargetSOAString(contents)
	case "SPF", "TXT":
		if txtFn == nil {
			return rc.SetTargetTXT(contents)
		}
		t, err := txtFn(contents)
		if err != nil {
			return fmt.Errorf("invalid TXT record: %s", contents)
		}
		return rc.SetTargetTXT(t)
	case "LUA":
		luaType, payload := ParseLuaContent(contents)
		rc.LuaRType = luaType
		if txtFn != nil {
			value, err := txtFn(payload)
			if err != nil {
				return fmt.Errorf("invalid LUA record: %s", contents)
			}
			return rc.SetTargetTXT(value)
		}
		value, err := DecodeLuaPayload(payload)
		if err != nil {
			return fmt.Errorf("invalid LUA record: %s", contents)
		}
		return rc.SetTargetTXT(value)
	case "SRV":
		return rc.SetTargetSRVString(contents)
	case "SSHFP":
		return rc.SetTargetSSHFPString(contents)
	case "SVCB", "HTTPS":
		return rc.SetTargetSVCBString(origin, contents)
	case "TLSA":
		return rc.SetTargetTLSAString(contents)
	default:
		// return fmt.Errorf("unknown rtype (%s) when parsing (%s) domain=(%s)", rtype, contents, origin)
		return MakeUnknown(rc, rtype, contents, origin)
	}
}

// PopulateFromString populates a RecordConfig given a type and string.  Many
// providers give all the parameters of a resource record in one big string.
// This helper function lets you not re-invent the wheel.
//
// NOTE: You almost always want to special-case TXT records. Every provider
// seems to quote them differently.
//
// Recommended calling convention:  Process the exceptions first, then use the
// PopulateFromString function for everything else.
//
//	 rtype := FILL_IN_TYPE
//		var err error
//	 rc := &models.RecordConfig{Type: rtype}
//	 rc.SetLabelFromFQDN(FILL_IN_NAME, origin)
//	 rc.TTL = uint32(FILL_IN_TTL)
//	 rc.Original = FILL_IN_ORIGINAL // The raw data received from provider (if needed later)
//		switch rtype {
//		case "MX":
//			// MX priority in a separate field.
//			err = rc.SetTargetMX(cr.Priority, target)
//		case "TXT":
//			// TXT records are stored verbatim; no quoting/escaping to parse.
//			err = rc.SetTargetTXT(target)
//		default:
//			err = rc.PopulateFromString(rtype, target, origin)
//		}
//		if err != nil {
//			return nil, fmt.Errorf("unparsable record type=%q received from PROVDER_NAME: %w", rtype, err)
//		}
//		return rc, nil
func (rc *RecordConfig) PopulateFromString(rtype, contents, origin string) error {
	if rc.Type != "" && rc.Type != rtype {
		panic(fmt.Errorf("assertion failed: rtype already set (%s) (%s)", rtype, rc.Type))
	}
	switch rc.Type = rtype; rtype { // #rtype_variations
	case "A":
		ip, err := netip.ParseAddr(contents)
		if err != nil || !ip.Is4() {
			return fmt.Errorf("invalid IP in A record: %s", contents)
		}
		return rc.SetTargetIP(ip) // Reformat to canonical form.
	case "AAAA":
		ip, err := netip.ParseAddr(contents)
		if err != nil || !ip.Is6() {
			return fmt.Errorf("invalid IP in AAAA record: %s", contents)
		}
		return rc.SetTargetIP(ip) // Reformat to canonical form.
	case "AKAMAICDN", "AKAMAITLC", "ALIAS", "ANAME", "CNAME", "NS", "PTR":
		return rc.SetTarget(contents)
	case "CAA":
		return rc.SetTargetCAAString(contents)
	case "DS":
		return rc.SetTargetDSString(contents)
	case "DNSKEY":
		return rc.SetTargetDNSKEYString(contents)
	case "DHCID":
		return rc.SetTarget(contents)
	case "DNAME":
		return rc.SetTarget(contents)
	case "LOC":
		return rc.SetTargetLOCString(origin, contents)
	case "MX":
		return rc.SetTargetMXString(contents)
	case "NAPTR":
		return rc.SetTargetNAPTRString(contents)
	case "OPENPGPKEY":
		return rc.SetTarget(contents)
	case "SMIMEA":
		return rc.SetTargetSMIMEAString(contents)
	case "SOA":
		return rc.SetTargetSOAString(contents)
	case "SPF", "TXT":
		return rc.SetTargetTXTs(ParseQuotedTxt(contents))
	case "LUA":
		luaType, payload := ParseLuaContent(contents)
		rc.LuaRType = luaType
		value, err := DecodeLuaPayload(payload)
		if err != nil {
			return fmt.Errorf("invalid LUA record: %s", contents)
		}
		return rc.SetTargetTXT(value)
	case "SRV":
		return rc.SetTargetSRVString(contents)
	case "SSHFP":
		return rc.SetTargetSSHFPString(contents)
	case "SVCB", "HTTPS":
		return rc.SetTargetSVCBString(origin, contents)
	case "TLSA":
		return rc.SetTargetTLSAString(contents)
	default:
		return fmt.Errorf("unknown rtype (%s) when parsing (%s) domain=(%s)",
			rtype, contents, origin)
	}
}
