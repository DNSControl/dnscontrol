package models

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
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

	typeNum, ok := dnsv2.StringToType[rtype]
	if !ok {
		return MakeUnknown(rc, rtype, contents, origin)
	}

	// Treat SPF as TXT.
	if typeNum == dnsv2.TypeSPF {
		typeNum = dnsv2.TypeTXT
	}

	// Use txtFn if provided to parse TXT records.
	if typeNum == dnsv2.TypeTXT {
		rc.TypeNum = dnsv2.TypeTXT
		rc.Type = "TXT"
		if txtFn != nil {
			var err error
			contents, err = txtFn(contents)
			if err != nil {
				return fmt.Errorf("invalid TXT record: %s", contents)
			}
		}
		rd := dnsrdatav2.TXT{Txt: []string{contents}}
		rc.SetRDATA(rd)

		// Populate legacy fields for backwards compatibility.
		return rc.SetTargetTXT(contents)
	}

	if typeNum == privatetypes.TypeLUA {
		luaType, payload := ParseLuaContent(contents)
		rc.LuaRType = luaType
		value, err := DecodeLuaPayload(payload)
		if err != nil {
			return fmt.Errorf("invalid LUA record: %s", contents)
		}
		err = rc.SetTargetTXT(value)
		if err != nil {
			return err
		}
		rd := privatetypesrdata.LUA{LuaType: luaType, LuaPayload: value}
		rc.SetRDATA(rd)
		return nil
	}

	if typeNum == privatetypes.TypeALIAS {
		// ALIAS is a private type: its rdata is a single hostname target. The
		// dnsv2 presentation parser has no parser for private types and would
		// treat the target as RFC3597 rdata, so set the target directly and
		// derive the V3 fields (.RDATA and .ComparableV3) via FixUp.
		rc.TypeNum = typeNum
		rc.Type = "ALIAS"
		if err := rc.SetTarget(contents); err != nil {
			return err
		}
		rc.RecomputeV3Fields(origin)
		return nil
	}

	return legacySetTargetParse(rc, typeNum, contents)

}

// PopulateFromString populates a RecordConfig given a type and string.  See PopulateFromStringFunc() for details.
func (rc *RecordConfig) PopulateFromString(rtype, contents, origin string) error {
	return rc.PopulateFromStringFunc(rtype, contents, origin, nil)
}
