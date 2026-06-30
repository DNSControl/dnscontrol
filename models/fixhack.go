package models

import (
	"fmt"
	"strings"

	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"

	_ "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes"
)

// RecomputeV3Fields re-derives the cached "V3 Fields" (.RDATA and
// .ComparableV3) after a record's underlying fields have been mutated after the
// record was first constructed (for example, a provider filling in a default
// value such as an R53_ALIAS zone_id). FixUp only populates these fields when
// they are empty, so they must be cleared first; otherwise the diff engine
// (which compares .ComparableV3) would keep seeing the pre-mutation values and
// report a spurious change.
func (rc *RecordConfig) RecomputeV3Fields(origin string) {
	rc.ComparableV3 = ""
	rc.ClearRDATA()
	rc.FixUp(origin)
}

// FixUp populates the "V3 Fields": .TypeNum, .RDATA and .ComparableV3.
func (rc *RecordConfig) FixUp(origin string) {

	switch rc.Type {
	case "IGNORE":
		return
	}

	// TypeNum:
	if rc.TypeNum == 0 && rc.Type != "ALIAS" && rc.Type != "IMPORT_TRANSFORM" {
		var err error
		tn, err := dnsutilv2.StringToType(rc.Type)
		if err != nil {
			s := fmt.Sprintf("BUG: FixUp: Unknown type %s", rc.Type)
			fmt.Println(s)
			panic(s)
		}
		rc.TypeNum = tn
	}

	// Populate .RDATA if needed:
	if rc.GetRDATA() == nil {

		//var err error
		switch rc.Type {

		case "BUNNY_DNS_PZ":
			rc.SetRDATA(&privatetypesrdata.BUNNYDNSPZ{})
		case "LUA":
			rc.SetRDATA(&privatetypesrdata.LUA{})
		case "CLOUDFLAREAPI_SINGLE_REDIRECT":
			rc.SetRDATA(&privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT{})
		case "CLOUDNS_WR":
			rc.SetRDATA(&privatetypesrdata.CLOUDNSWR{})
		case "NETLIFY":
			rc.SetRDATA(&privatetypesrdata.NETLIFY{})
		case "NETLIFYV6":
			rc.SetRDATA(&privatetypesrdata.NETLIFYV6{})
		case "AKAMAICDN":
			rc.SetRDATA(&privatetypesrdata.AKAMAICDN{})
		case "AKAMAITLC":
			rc.SetRDATA(&privatetypesrdata.AKAMAITLC{})
		case "BUNNY_DNS_RDR":
			rc.SetRDATA(&privatetypesrdata.BUNNYDNSRDR{})
		case "IMPORT_TRANSFORM":
			rc.ClearRDATA()

		case "A":
			rd, err := MakeA(origin, nil, rc.GetTargetIP())
			errorChk(err)
			rc.SetRDATA(rd)
		case "ALIAS":
			rd, err := privatetypesrdata.MakeALIAS(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "AAAA":
			rd, err := MakeAAAA(origin, nil, rc.GetTargetIP())
			errorChk(err)
			rc.SetRDATA(rd)
		case "ADGUARDHOME_A_PASSTHROUGH":
			rd, err := privatetypesrdata.MakeADGUARDHOMEAPASSTHROUGH(origin, nil)
			errorChk(err)
			rc.SetRDATA(rd)
		case "ADGUARDHOME_AAAA_PASSTHROUGH":
			rd, err := privatetypesrdata.MakeADGUARDHOMEAAAAPASSTHROUGH(origin, nil)
			errorChk(err)
			rc.SetRDATA(rd)
		case "AZURE_ALIAS":
			rd, err := privatetypesrdata.MakeAZUREALIAS(origin, nil, rc.AzureAlias["type"], rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)

		case "CAA":
			rd, err := MakeCAA(origin, nil, rc.CaaFlag, rc.CaaTag, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "CNAME":
			rd, err := MakeCNAME(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "CF_WORKER_ROUTE":
			part := strings.SplitN(rc.GetTargetField(), ",", 2)
			rd, err := privatetypesrdata.MakeCFWORKERROUTE(origin, nil, part[0], part[1])
			errorChk(err)
			rc.SetRDATA(rd)

		case "DHCID":
			rd, err := MakeDHCID(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "DNAME":
			rd, err := MakeDNAME(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "DNSKEY":
			rd, err := MakeDNSKEY(origin, nil, rc.DnskeyFlags, rc.DnskeyProtocol, rc.DnskeyAlgorithm, rc.DnskeyPublicKey)
			errorChk(err)
			rc.SetRDATA(rd)
		case "DS":
			rd, err := MakeDS(origin, nil, rc.DsKeyTag, rc.DsAlgorithm, rc.DsDigestType, rc.DsDigest)
			errorChk(err)
			rc.SetRDATA(rd)

		case "FRAME":
			rd, err := privatetypesrdata.MakeFRAME(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)

		case "HTTPS":
			rd, err := MakeHTTPS(origin, nil, rc.SvcPriority, rc.GetTargetField(), rc.SvcParams)
			if err != nil {
				s := fmt.Sprintf("BUG: FixUp: MakeHTTPS failed for record %s IN %s %s: %v", rc.NameFQDN, rc.Type, rc.GetTargetField(), err)
				fmt.Println(s)
				panic(s)
			}
			rc.SetRDATA(rd)

		case "LOC":
			rd, err := MakeLOC(origin, nil, rc.LocVersion, rc.LocSize, rc.LocHorizPre, rc.LocVertPre, rc.LocLatitude, rc.LocLongitude, rc.LocAltitude)
			errorChk(err)
			rc.SetRDATA(rd)

		case "MIKROTIK_FWD":
			rd, err := privatetypesrdata.MakeMIKROTIKFWD(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "MIKROTIK_NXDOMAIN":
			rd, err := privatetypesrdata.MakeMIKROTIKNXDOMAIN(origin, nil)
			errorChk(err)
			rc.SetRDATA(rd)
		case "MX":
			rd, err := MakeMX(origin, nil, rc.MxPreference, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)

		case "NS":
			rd, err := MakeNS(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "NAPTR":
			rd, err := MakeNAPTR(origin, nil, rc.NaptrOrder, rc.NaptrPreference, rc.NaptrFlags, rc.NaptrService, rc.NaptrRegexp, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)

		case "OPENPGPKEY":
			rd, err := MakeOPENPGPKEY(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)

		case "PORKBUN_URLFWD":
			// NB(tlim): Should this use privatetypesrdata.MakePORKBUNURLFWD() instead?
			rc.SetRDATA(&privatetypesrdata.PORKBUNURLFWD{
				Target:      rc.GetTargetField(),
				TypeName:    rc.Metadata["type"],
				IncludePath: rc.Metadata["includePath"],
				Wildcard:    rc.Metadata["wildcard"],
			})

		case "PTR":
			rd, err := MakePTR(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)

		case "RP":
			//rd, err = MakeRP(origin, rc.F.(dnsv1.RP).Mbox, rc.F.(dnsv1.RP).Txt)
			// RP is native to RecordConfigV3. No FixUP is needed or possible.
		case "R53_ALIAS":
			rd, err := privatetypesrdata.MakeR53ALIAS(origin, nil,
				rc.R53Alias["type"],
				rc.GetTargetField(),
				rc.R53Alias["zone_id"],
				rc.R53Alias["evaluate_target_health"],
			)
			errorChk(err)
			rc.SetRDATA(rd)

		case "SMIMEA":
			rd, err := MakeSMIMEA(origin, nil, rc.SmimeaUsage, rc.SmimeaSelector, rc.SmimeaMatchingType, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "SOA":
			rd, err := MakeSOA(origin, nil, rc.GetTargetField(), rc.SoaMbox, rc.SoaSerial, rc.SoaRefresh, rc.SoaRetry, rc.SoaExpire, rc.SoaMinttl)
			errorChk(err)
			rc.SetRDATA(rd)
		case "SRV":
			rd, err := MakeSRV(origin, nil, rc.SrvPriority, rc.SrvWeight, rc.SrvPort, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "SSHFP":
			rd, err := MakeSSHFP(origin, nil, rc.SshfpAlgorithm, rc.SshfpFingerprint, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "SVCB":
			rd, err := MakeSVCB(origin, nil, rc.SvcPriority, rc.GetTargetField(), rc.SvcParams)
			errorChk(err)
			rc.SetRDATA(rd)

		case "TLSA":
			rd, err := MakeTLSA(origin, nil, rc.TlsaUsage, rc.TlsaSelector, rc.TlsaMatchingType, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)
		case "TXT":
			rd, err := MakeTXT(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)

		case "URL":
			rd, err := privatetypesrdata.MakeURL(origin, nil,
				rc.GetTargetField(),
				rc.Metadata["includePath"],
				rc.Metadata["wildcard"],
			)
			errorChk(err)
			rc.SetRDATA(rd)
		case "URL301":
			rd, err := privatetypesrdata.MakeURL301(origin, nil, rc.GetTargetField())
			errorChk(err)
			rc.SetRDATA(rd)

		default:
			panic(fmt.Sprintf("RDATA FIXUP NOT IMPLEMENTED TYPE=%q", rc.Type))
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
			rd := rc.GetRDATA().(dnsrdatav2.SOA)
			rc.ComparableV3 = fmt.Sprintf("%s %s X %d %d %d %d", rd.Ns, rd.Mbox, rd.Refresh, rd.Retry, rd.Expire, rd.Minttl)

		default:
			if rc.GetRDATA() == nil {
				panic(fmt.Sprintf("BUG: FixUp: .RDATA is nil for type %s", rc.Type))
			}
			rc.ComparableV3 = strings.TrimSpace(rc.GetRDATA().String())
		}
	}
}

func errorChk(err error) {
	if err == nil {
		return
	}
	fmt.Printf("BUG: FixUp: Make$TYPE() failed: %v\n", err)
}
