package models

import (
	"fmt"
	"strings"

	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
	dnsv1 "github.com/miekg/dns"

	_ "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes"
)

// FixUp populates the "V3 Fields": .TypeNum, .RDATA and .ComparableV3.
func (rc *RecordConfig) FixUp(origin string) {

	switch rc.Type {
	case "IGNORE":
		return
	}

	// TypeNum:
	if rc.TypeNum == 0 && rc.Type != "ALIAS" {
		var err error
		tn, err := dnsutilv2.StringToType(rc.Type)
		if err != nil {
			panic(fmt.Sprintf("BUG: FixUp: Unknown type %s", rc.Type))
		}
		rc.TypeNum = tn
	}

	// Populate .RDATA if needed:
	if rc.RDATA == nil {

		var err error
		switch rc.Type {

		// Incomplete
		// case "PORKBUN_URLFWD":
		// 	rc.RDATA = privatetypesrdata.PORKBUN_URLFWD{}
		case "BUNNY_DNS_PZ":
			rc.RDATA = privatetypesrdata.BUNNYDNSPZ{}
		case "LUA":
			rc.RDATA = privatetypesrdata.LUA{}
		case "CLOUDFLAREAPI_SINGLE_REDIRECT":
			rc.RDATA = privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT{}
		case "CLOUDNS_WR":
			rc.RDATA = privatetypesrdata.CLOUDNSWR{}
		case "NETLIFY":
			rc.RDATA = privatetypesrdata.NETLIFY{}
		case "NETLIFYV6":
			rc.RDATA = privatetypesrdata.NETLIFYV6{}
		case "AKAMAICDN":
			rc.RDATA = privatetypesrdata.AKAMAICDN{}
		case "AKAMAITLC":
			rc.RDATA = privatetypesrdata.AKAMAITLC{}
		case "BUNNY_DNS_RDR":
			rc.RDATA = privatetypesrdata.BUNNYDNSRDR{}

		case "A":
			rc.RDATA, err = MakeA(origin, rc.GetTargetIP())
		case "ALIAS":
			rc.RDATA, err = privatetypesrdata.MakeALIAS(origin, rc.GetTargetField())
		case "AAAA":
			rc.RDATA, err = MakeAAAA(origin, rc.GetTargetIP())
		case "ADGUARDHOME_A_PASSTHROUGH":
			rc.RDATA, err = privatetypesrdata.MakeADGUARDHOMEAPASSTHROUGH(origin)
		case "ADGUARDHOME_AAAA_PASSTHROUGH":
			rc.RDATA, err = privatetypesrdata.MakeADGUARDHOMEAAAAPASSTHROUGH(origin)
		case "AZURE_ALIAS":
			rc.RDATA, err = privatetypesrdata.MakeAZUREALIAS(origin, rc.AzureAlias["type"], rc.GetTargetField())

		case "CAA":
			rc.RDATA, err = MakeCAA(origin, rc.CaaFlag, rc.CaaTag, rc.GetTargetField())
		case "CNAME":
			rc.RDATA, err = MakeCNAME(origin, rc.GetTargetField())
		case "CF_WORKER_ROUTE":
			part := strings.SplitN(rc.GetTargetField(), ",", 2)
			rc.RDATA, err = privatetypesrdata.MakeCFWORKERROUTE(origin, part[0], part[1])

		case "DHCID":
			rc.RDATA, err = MakeDHCID(origin, rc.GetTargetField())
		case "DNAME":
			rc.RDATA, err = MakeDNAME(origin, rc.GetTargetField())
		case "DNSKEY":
			rc.RDATA, err = MakeDNSKEY(origin, rc.DnskeyFlags, rc.DnskeyProtocol, rc.DnskeyAlgorithm, rc.DnskeyPublicKey)
		case "DS":
			rc.RDATA, err = MakeDS(origin, rc.DsKeyTag, rc.DsAlgorithm, rc.DsDigestType, rc.DsDigest)

		case "FRAME":
			rc.RDATA, err = privatetypesrdata.MakeFRAME(origin, rc.GetTargetField())

		case "HTTPS":
			rd, err := MakeHTTPS(origin, rc.SvcPriority, rc.GetTargetField(), rc.SvcParams)
			if err != nil {
				panic(fmt.Sprintf("BUG: FixUp: MakeHTTPS failed for record %s IN %s %s: %v", rc.NameFQDN, rc.Type, rc.GetTargetField(), err))
			}
			rc.RDATA = rd

		case "LOC":
			rc.RDATA, err = MakeLOC(origin, rc.LocVersion, rc.LocSize, rc.LocHorizPre, rc.LocVertPre, rc.LocLatitude, rc.LocLongitude, rc.LocAltitude)

		case "MIKROTIK_FWD":
			rc.RDATA, err = privatetypesrdata.MakeMIKROTIKFWD(origin, rc.GetTargetField())
		case "MIKROTIK_NXDOMAIN":
			rc.RDATA, err = privatetypesrdata.MakeMIKROTIKNXDOMAIN(origin)
		case "MX":
			rc.RDATA, err = MakeMX(origin, rc.MxPreference, rc.GetTargetField())

		case "NS":
			rc.RDATA, err = MakeNS(origin, rc.GetTargetField())
		case "NAPTR":
			rc.RDATA, err = MakeNAPTR(origin, rc.NaptrOrder, rc.NaptrPreference, rc.NaptrFlags, rc.NaptrService, rc.NaptrRegexp, rc.GetTargetField())

		case "OPENPGPKEY":
			rc.RDATA, err = MakeOPENPGPKEY(origin, rc.GetTargetField())

		case "PORKBUN_URLFWD":
			rc.RDATA, err = privatetypesrdata.MakePORKBUNURLFWD(origin, rc.GetTargetField())

		case "PTR":
			rc.RDATA, err = MakePTR(origin, rc.GetTargetField())

		case "RP":
			rc.RDATA, err = MakeRP(origin, rc.F.(dnsv1.RP).Mbox, rc.F.(dnsv1.RP).Txt)
		case "R53_ALIAS":
			rc.RDATA, err = privatetypesrdata.MakeR53ALIAS(origin, rc.R53Alias["type"], rc.GetTargetField(), rc.R53Alias["zone_id"], rc.R53Alias["evaluate_target_health"])

		case "SMIMEA":
			rc.RDATA, err = MakeSMIMEA(origin, rc.SmimeaUsage, rc.SmimeaSelector, rc.SmimeaMatchingType, rc.GetTargetField())
		case "SOA":
			rc.RDATA, err = MakeSOA(origin, rc.GetTargetField(), rc.SoaMbox, rc.SoaSerial, rc.SoaRefresh, rc.SoaRetry, rc.SoaExpire, rc.SoaMinttl)
		case "SRV":
			rc.RDATA, err = MakeSRV(origin, rc.SrvPriority, rc.SrvWeight, rc.SrvPort, rc.GetTargetField())
		case "SSHFP":
			rc.RDATA, err = MakeSSHFP(origin, rc.SshfpAlgorithm, rc.SshfpFingerprint, rc.GetTargetField())
		case "SVCB":
			rc.RDATA, err = MakeSVCB(origin, rc.SvcPriority, rc.GetTargetField(), rc.SvcParams)

		case "TLSA":
			rc.RDATA, err = MakeTLSA(origin, rc.TlsaUsage, rc.TlsaSelector, rc.TlsaMatchingType, rc.GetTargetField())
		case "TXT":
			rc.RDATA, err = MakeTXT(origin, rc.GetTargetField())

		case "URL":
			rc.RDATA, err = privatetypesrdata.MakeURL(origin,
				rc.GetTargetField(),
				rc.Metadata["includePath"],
				rc.Metadata["wildcard"],
			)
		case "URL301":
			rc.RDATA, err = privatetypesrdata.MakeURL(origin, rc.GetTargetField())

		default:
			panic(fmt.Sprintf("RDATA FIXUP NOT IMPLEMENTED TYPE=%q", rc.Type))
		}
		if err != nil {
			panic(fmt.Sprintf("BUG: FixUp: Make%s( failed for record %s IN %s %s: %v", rc.Type, rc.NameFQDN, rc.Type, rc.GetTargetField(), err))
		}
	}

	// .ComparableV3:
	if rc.ComparableV3 == "" {
		switch rc.Type {
		case "SOA":
			// The comparable string for SOA intentionally excludes the serial
			// number, because the serial number changes on every update and
			// would prevent correct diffing. List it as "X" so-as it stands out
			// in debug output that the serial is intentionally excluded.
			rc.ComparableV3 = fmt.Sprintf("%s %s X %d %d %d %d", rc.GetTargetField(), rc.SoaMbox, rc.SoaRefresh, rc.SoaRetry, rc.SoaExpire, rc.SoaMinttl)
		// case "HTTPS", "SVCB":
		// 	x := rc.RDATA.String()
		// 	x = strings.TrimSpace(x)
		// 	rc.ComparableV3 = x
		default:
			if rc.RDATA == nil {
				panic(fmt.Sprintf("BUG: FixUp: .RDATA is nil for type %s", rc.Type))
			}
			x := rc.RDATA.String()
			x = strings.TrimSpace(x)
			rc.ComparableV3 = x
		}

		// Note to self: RDATA.String() sometimes leaves a trailing space.  File a bug.
		// if strings.HasSuffix(rc.ComparableV3, " ") {
		// 	rc.ComparableV3 = rc.ComparableV3 + "W"
		// }
	}
}
