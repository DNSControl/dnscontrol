package models

import (
	"fmt"

	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

func (rc *RecordConfig) copyRDtoLegacyFields() error {
	// Hack to back-fill legacy fields. This will go away eventually.
	switch rd := rc.GetRDATA().(type) {
	case privatetypesrdata.ADGUARDHOMEAPASSTHROUGH:
		rc.SetTarget(rd.Target)
	case privatetypesrdata.ADGUARDHOMEAAAAPASSTHROUGH:
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
		rc.SetTarget(fmt.Sprintf("%s,%s", rd.When, rd.Then))
	case privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT:
		// no-op
	case dnsrdatav2.CNAME:
		rc.SetTarget(rd.Target)

	case dnsrdatav2.DHCID:
		rc.SetTarget(rd.Digest)
	case dnsrdatav2.DNAME:
		rc.SetTarget(rd.Target)
	case dnsrdatav2.DS:
		rc.DsKeyTag, rc.DsAlgorithm, rc.DsDigestType, rc.DsDigest = rd.KeyTag, rd.Algorithm, rd.DigestType, rd.Digest
	case dnsrdatav2.DNSKEY:
		rc.DnskeyFlags, rc.DnskeyProtocol, rc.DnskeyAlgorithm, rc.DnskeyPublicKey = rd.Flags, rd.Protocol, rd.Algorithm, rd.PublicKey
	case privatetypesrdata.FRAME:
		rc.SetTarget(rd.Target)

	case dnsrdatav2.LOC:
		rc.SetTargetLOC(rd.Version, rd.Latitude, rd.Longitude, rd.Altitude, rd.Size, rd.HorizPre, rd.VertPre)
	case privatetypesrdata.LUA:
		rc.LuaRType = rd.LuaType
		rc.SetTarget(rd.LuaPayload)

	case privatetypesrdata.MIKROTIKFWD:
		rc.SetTarget(rd.ForwardTo)
	case privatetypesrdata.MIKROTIKNXDOMAIN:
		// no-op
	case dnsrdatav2.MX:
		rc.SetTargetMX(rd.Preference, rd.Mx)

	case dnsrdatav2.NAPTR:
		rc.SetTargetNAPTR(rd.Order, rd.Preference, rd.Flags, rd.Service, rd.Regexp, rd.Replacement)
	case dnsrdatav2.NS:
		rc.SetTarget(rd.Ns)

	case dnsrdatav2.OPENPGPKEY:
		rc.SetTarget(rd.PublicKey)

	case privatetypesrdata.PORKBUNURLFWD:
		if rc.Metadata == nil {
			rc.Metadata = map[string]string{}
		}
		rc.Metadata["type"] = rd.TypeName
		rc.Metadata["includePath"] = rd.IncludePath
		rc.Metadata["wildcard"] = rd.Wildcard
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
		rc.SetTargetSMIMEA(rd.Usage, rd.Selector, rd.MatchingType, rd.Certificate)
	case dnsrdatav2.SOA:
		rc.SetTargetSOA(rd.Ns, rd.Mbox, rd.Serial, rd.Refresh, rd.Retry, rd.Expire, rd.Minttl)
	case dnsrdatav2.SRV:
		rc.SetTargetSRV(rd.Priority, rd.Weight, rd.Port, rd.Target)
	case dnsrdatav2.SSHFP:
		rc.SetTargetSSHFP(rd.Algorithm, rd.Type, rd.FingerPrint)
	case dnsrdatav2.SVCB: // There is no dnsrdatav2.HTTPS
		rc.SvcPriority = rd.Priority
		rc.SetTarget(rd.Target)
		rc.SvcParams = svcbv2ValueToString(rd.Value)

	case dnsrdatav2.TLSA:
		rc.SetTargetTLSA(rd.Usage, rd.Selector, rd.MatchingType, rd.Certificate)
	case dnsrdatav2.TXT:
		rc.SetTargetTXTs(rd.Txt)

	case privatetypesrdata.URL:
		rc.SetTarget(rd.Location)
		if rc.Metadata == nil {
			rc.Metadata = map[string]string{}
		}
		rc.Metadata["includePath"] = fmt.Sprintf("%t", rd.PorkbunIncludePath)
		rc.Metadata["wildcard"] = fmt.Sprintf("%t", rd.PorkbunWildCard)
	case privatetypesrdata.URL301:
		rc.SetTarget(rd.Location)

	default:
		return fmt.Errorf("assertion failed: copyRDtoLegacyFields has not implemented type %T", rd)
	}

	return nil
}
