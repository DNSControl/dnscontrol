package models

import (
	"fmt"

	"codeberg.org/miekg/dns/rdata"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

func backfill(rc *RecordConfig) error {
	// Hack to back-fill legacy fields. This will go away eventually.
	switch rd := rc.GetRDATA().(type) {
	case privatetypesrdata.ALIAS:
		rc.SetTarget(rd.Target)
	case privatetypesrdata.AZUREALIAS:
		rc.SetTarget(rd.Target)
		rc.AzureAlias = map[string]string{"type": rd.AliasType}
	case privatetypesrdata.ADGUARDHOMEAPASSTHROUGH:
		rc.SetTarget(rd.Target)
	case privatetypesrdata.ADGUARDHOMEAAAAPASSTHROUGH:
		rc.SetTarget(rd.Target)
	case dnsrdatav2.A:
		rc.SetTargetIP(rd.Addr)
	case dnsrdatav2.AAAA:
		rc.SetTargetIP(rd.Addr)
	case privatetypesrdata.BUNNYDNSPZ:
		//rc.SetTarget(fmt.Sprintf("%v", rd.PullZoneID))
		// no-op
	case dnsrdatav2.CAA:
		rc.SetTargetCAA(rd.Flag, rd.Tag, rd.Value)
	case privatetypesrdata.CFWORKERROUTE:
		rc.SetTarget(fmt.Sprintf("%s,%s", rd.When, rd.Then))
	case dnsrdatav2.CNAME:
		rc.SetTarget(rd.Target)
	case dnsrdatav2.DHCID:
		rc.SetTarget(rd.Digest)
	case dnsrdatav2.DNAME:
		rc.SetTarget(rd.Target)
	case dnsrdatav2.DS:
		rc.SetTargetDS(rd.KeyTag, rd.Algorithm, rd.DigestType, rd.Digest)
	case dnsrdatav2.DNSKEY:
		rc.SetTargetDNSKEY(rd.Flags, rd.Protocol, rd.Algorithm, rd.PublicKey)
	case privatetypesrdata.FRAME:
		rc.SetTarget(rd.Target)
	case dnsrdatav2.LOC:
		rc.SetTargetLOC(rd.Version, rd.Latitude, rd.Longitude, rd.Altitude, rd.Size, rd.HorizPre, rd.VertPre)
	case privatetypesrdata.MIKROTIKFWD:
		rc.SetTarget(rd.ForwardTo)
	case privatetypesrdata.MIKROTIKNXDOMAIN:
		// no-op
	case dnsrdatav2.MX:
		rc.SetTargetMX(rd.Preference, rd.Mx)
	case dnsrdatav2.NS:
		rc.SetTarget(rd.Ns)
	case dnsrdatav2.NAPTR:
		rc.SetTargetNAPTR(rd.Order, rd.Preference, rd.Flags, rd.Service, rd.Regexp, rd.Replacement)
	case rdata.OPENPGPKEY:
		rc.SetTarget(rd.PublicKey)
	case dnsrdatav2.PTR:
		rc.SetTarget(rd.Ptr)
	case dnsrdatav2.RP:
		// noop -- no legacy fields
	case dnsrdatav2.SMIMEA:
		rc.SetTargetSMIMEA(rd.Usage, rd.Selector, rd.MatchingType, rd.Certificate)
	case dnsrdatav2.SOA:
		rc.SetTargetSOA(rd.Ns, rd.Mbox, rd.Serial, rd.Refresh, rd.Retry, rd.Expire, rd.Minttl)
	case dnsrdatav2.SRV:
		rc.SetTargetSRV(rd.Priority, rd.Weight, rd.Port, rd.Target)
	case dnsrdatav2.SVCB: // There is no dnsrdatav2.HTTPS
		rc.SvcPriority = rd.Priority
		rc.SetTarget(rd.Target)
		rc.SvcParams = svcbv2ValueToString(rd.Value)
	case dnsrdatav2.SSHFP:
		rc.SetTargetSSHFP(rd.Algorithm, rd.Type, rd.FingerPrint)
	case dnsrdatav2.TLSA:
		rc.SetTargetTLSA(rd.Usage, rd.Selector, rd.MatchingType, rd.Certificate)
	case dnsrdatav2.TXT:
		rc.SetTargetTXTs(rd.Txt)
	default:
		switch rc.Type {
		case "CLOUDFLAREAPI_SINGLE_REDIRECT":
			// no-op
		case "PORKBUN_URLFWD":
			p := rd.(privatetypesrdata.PORKBUNURLFWD)
			if rc.Metadata == nil {
				rc.Metadata = map[string]string{}
			}
			rc.Metadata["type"] = p.TypeName
			rc.Metadata["includePath"] = p.IncludePath
			rc.Metadata["wildcard"] = p.Wildcard
		case "R53_ALIAS":
			p := rd.(privatetypesrdata.R53ALIAS)
			if rc.R53Alias == nil {
				rc.R53Alias = map[string]string{}
			}
			rc.R53Alias["type"] = p.AliasType
			rc.SetTarget(p.Target)
			rc.R53Alias["zone_id"] = p.ZoneID
			rc.R53Alias["evaluate_target_health"] = p.EvalTargetHealth

		case "URL":
			u := rd.(privatetypesrdata.URL)
			rc.SetTarget(u.Location)
			if rc.Metadata == nil {
				rc.Metadata = map[string]string{}
			}
			rc.Metadata["includePath"] = fmt.Sprintf("%t", u.PorkbunIncludePath)
			rc.Metadata["wildcard"] = fmt.Sprintf("%t", u.PorkbunWildCard)
		case "URL301":
			u := rd.(privatetypesrdata.URL301)
			rc.SetTarget(u.Location)
		case "SVCB":
			// skip
		default:
			return fmt.Errorf("assertion failed: NewRecordConfig back-fill has not implemented type %T", rd)
			// TODO:
			//case privatetypes..AzureAlias:
			//case privatetypes..LUA:
			//case privatetypes..R53Alias:
			//case privatetypes..AKAMAITLC:
		}
	}
	return nil
}
