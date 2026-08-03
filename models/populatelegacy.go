package models

/*
This file supports the backwards-compatibility mode while we are converting to
RecordConfig V3.  In particular, it contains the `copyRDtoLegacyFields()`
function, which copies individual fields from rc.rdata to the old, legacy
fields.
*/

import (
	"fmt"

	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
)

// copyRDtoLegacyFields copies the fields from rc.rdata to the legacy fields for that record type.
// This will go away when the migration to RecordConfig V3 is complete.
// For newer record types, this function will be a no-op as they have no legacy fields.
func (rc *RecordConfig) copyRDtoLegacyFields() error {
	// Hack to back-fill legacy fields. This will go away eventually.
	switch rd := rc.GetRDATA().(type) {
	case privatetypesrdata.ADGUARDHOMEAPASSTHROUGH:
		rc.SetTarget(rd.Target)
	case privatetypesrdata.ADGUARDHOMEAAAAPASSTHROUGH:
		rc.SetTarget(rd.Target)
	case privatetypesrdata.AKAMAICDN:
		rc.SetTarget(rd.Target)
	case privatetypesrdata.AKAMAITLC:
		rc.AnswerType = rd.AnswerType
		rc.SetTarget(rd.Target)
	case privatetypesrdata.ALIAS:
		rc.SetTarget(rd.Target)
	case privatetypesrdata.AZUREALIAS:
		rc.SetTarget(rd.Target)
		rc.AzureAlias = map[string]string{"type": rd.AliasType}
	case dnsrdatav2.A:
		rc.SetTargetIP(rd.Addr)
	case dnsrdatav2.AAAA:
		rc.SetTargetIP(rd.Addr)

	case privatetypesrdata.BUNNYDNSPZ:
		// no-op

	case dnsrdatav2.CAA:
		rc.CaaFlag = rd.Flag
		rc.CaaTag = rd.Tag
		rc.SetTarget(rd.Value)
	case privatetypesrdata.CFWORKERROUTE:
		// no-op
	case privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT:
		// no-op
	case privatetypesrdata.CLOUDNSWR:
		rc.SetTarget(rd.Target)
	case dnsrdatav2.CNAME:
		rc.SetTarget(rd.Target)

	case dnsrdatav2.DHCID:
		rc.SetTarget(rd.Digest)
	case dnsrdatav2.DNAME:
		rc.SetTarget(rd.Target)
	case dnsrdatav2.DS:
		rc.DsKeyTag, rc.DsAlgorithm, rc.DsDigestType, rc.DsDigest = rd.KeyTag, rd.Algorithm, rd.DigestType, rd.Digest
	case dnsrdatav2.DNSKEY:
		// no-op
	case privatetypesrdata.FRAME:
		rc.SetTarget(rd.Target)

	case dnsrdatav2.LOC:
		// no-op
	case privatetypesrdata.LUA:
		rc.LuaRType = rd.LuaType
		rc.SetTarget(rd.LuaPayload)

	case privatetypesrdata.MIKROTIKFWD:
		// no-op
	case privatetypesrdata.MIKROTIKNXDOMAIN:
		// no-op
	case privatetypesrdata.MIKROTIKFORWARDER:
		// no-op
	case dnsrdatav2.MX:
		rc.MxPreference = rd.Preference
		rc.SetTarget(rd.Mx)

	case dnsrdatav2.NAPTR:
		rc.NaptrOrder, rc.NaptrPreference, rc.NaptrFlags, rc.NaptrService, rc.NaptrRegexp = rd.Order, rd.Preference, rd.Flags, rd.Service, rd.Regexp
		rc.SetTarget(rd.Replacement)

	case dnsrdatav2.NS:
		rc.SetTarget(rd.Ns)

	case dnsrdatav2.OPENPGPKEY:
		rc.SetTarget(rd.PublicKey)

	case privatetypesrdata.PORKBUNURLFWD:
		rc.SetTarget(rd.Location)
	case dnsrdatav2.PTR:
		rc.SetTarget(rd.Ptr)

	case dnsrdatav2.RP:
		// noop -- no legacy fields
	case privatetypesrdata.R53ALIAS:
		if rc.R53Alias == nil {
			rc.R53Alias = map[string]string{}
		}
		rc.R53Alias["type"] = rd.AliasType
		rc.SetTarget(rd.Target)
		if rd.ZoneID != "" {
			rc.R53Alias["zone_id"] = rd.ZoneID
		}
		rc.R53Alias["evaluate_target_health"] = rd.EvalTargetHealth

	case dnsrdatav2.SMIMEA:
		// no-op
	case dnsrdatav2.SOA:
		// noop -- no legacy fields
	case dnsrdatav2.SRV:
		rc.SrvPriority, rc.SrvWeight, rc.SrvPort = rd.Priority, rd.Weight, rd.Port
		rc.SetTarget(rd.Target)
	case dnsrdatav2.SSHFP:
		rc.SshfpAlgorithm = rd.Algorithm
		rc.SshfpFingerprint = rd.Type // Yes, all these years we've been storing things in the wrong field.
		rc.SetTarget(rd.FingerPrint)
	case dnsrdatav2.SVCB: // There is no dnsrdatav2.HTTPS
		rc.SvcPriority = rd.Priority
		rc.SetTarget(rd.Target)
		rc.SvcParams = Svcbv2ValueToString(rd.Value)

	case dnsrdatav2.TLSA:
		rc.TlsaUsage, rc.TlsaSelector, rc.TlsaMatchingType = rd.Usage, rd.Selector, rd.MatchingType
		rc.SetTarget(rd.Certificate)
	case dnsrdatav2.TXT:
		// TXT stores its value only in .rdata (the single source of truth).
		// The TXT accessors (GetTargetField/GetTargetTXTJoined/...) read it
		// from there; there is no legacy .target back-fill.

	case privatetypesrdata.URL:
		rc.SetTarget(rd.Location)
	case privatetypesrdata.URL301:
		rc.SetTarget(rd.Location)

	default:
		return fmt.Errorf("assertion failed: copyRDtoLegacyFields has not implemented type %T", rd)
	}

	return nil
}
