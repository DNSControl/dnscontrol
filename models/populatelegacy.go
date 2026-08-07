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

	case privatetypesrdata.AKAMAITLC:
		rc.AnswerType = rd.AnswerType
	case privatetypesrdata.AZUREALIAS:
		rc.AzureAlias = map[string]string{"type": rd.AliasType}
	case privatetypesrdata.LUA:
		rc.LuaRType = rd.LuaType
	case privatetypesrdata.R53ALIAS:
		if rc.R53Alias == nil {
			rc.R53Alias = map[string]string{}
		}
		rc.R53Alias["type"] = rd.AliasType
		if rd.ZoneID != "" {
			rc.R53Alias["zone_id"] = rd.ZoneID
		}
		rc.R53Alias["evaluate_target_health"] = rd.EvalTargetHealth
		// The alias target (DNSName) is the record's primary target. It must be
		// mirrored into the legacy .target field: copyLegacyFieldsToRD() re-derives
		// the RDATA from GetTargetField() (e.g. when RecomputeV3Fields() rebuilds
		// after a provider fills in zone_id), and without this the target would be
		// lost and default to the zone apex.
		if err := rc.SetTarget(rd.Target); err != nil {
			return err
		}

	case dnsrdatav2.A:
	case dnsrdatav2.AAAA:
	case dnsrdatav2.CAA:
	case dnsrdatav2.CNAME:
	case dnsrdatav2.DHCID:
	case dnsrdatav2.DNAME:
	case dnsrdatav2.DNSKEY:
	case dnsrdatav2.DS:
	case dnsrdatav2.LOC:
	case dnsrdatav2.MX:
	case dnsrdatav2.NAPTR:
	case dnsrdatav2.NS:
	case dnsrdatav2.OPENPGPKEY:
	case dnsrdatav2.PTR:
	case dnsrdatav2.RP:
	case dnsrdatav2.SMIMEA:
	case dnsrdatav2.SOA:
	case dnsrdatav2.SRV:
	case dnsrdatav2.SSHFP:
	case dnsrdatav2.SVCB: // There is no dnsrdatav2.HTTPS
	case dnsrdatav2.TLSA:
	case dnsrdatav2.TXT:
	case privatetypesrdata.ADGUARDHOMEAAAAPASSTHROUGH:
	case privatetypesrdata.ADGUARDHOMEAPASSTHROUGH:
	case privatetypesrdata.AKAMAICDN:
	case privatetypesrdata.ALIAS:
	case privatetypesrdata.BUNNYDNSPZ:
	case privatetypesrdata.CFWORKERROUTE:
	case privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT:
	case privatetypesrdata.CLOUDNSWR:
	case privatetypesrdata.FRAME:
	case privatetypesrdata.MIKROTIKFORWARDER:
	case privatetypesrdata.MIKROTIKFWD:
	case privatetypesrdata.MIKROTIKNXDOMAIN:
	case privatetypesrdata.PORKBUNURLFWD:
	case privatetypesrdata.URL301:
	case privatetypesrdata.URL:

	default:
		return fmt.Errorf("assertion failed: copyRDtoLegacyFields has not implemented type %T", rd)
	}

	return nil
}
