package models

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v4/pkg/privatetypes"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// RecomputeV3Fields re-derives the cached "V3 Fields" (.RDATA and
// .ComparableV3) after a record's underlying fields have been mutated after the
// record was first constructed (for example, a provider filling in a default
// value such as an R53_ALIAS zone_id). FixUp only populates these fields when
// they are empty, so they must be cleared first; otherwise the diff engine
// (which compares .ComparableV3) would keep seeing the pre-mutation values and
// report a spurious change.
func (rc *RecordConfig) RecomputeV3Fields(origin string) {
	rc.ClearRDATA()
	rc.copyLegacyFieldsToRD(origin)
	rc.RegenerateComparableV3()
}

// FixRD populates the "V3 Fields": .TypeNum, .RDATA and .ComparableV3. It is non-destructive for .RDATA and .ComparableV3 if those are non-nil.
func (rc *RecordConfig) FixRD(origin string) {

	// fmt.Printf("DEBUG: FixRD: %s\n", rc.String())
	rc.fixTypeNum()

	// Populate .RDATA if needed:
	if rc.GetRDATA() == nil {
		rc.copyLegacyFieldsToRD(origin)
	}

	// Generate .ComparableV3 if empty:
	if rc.ComparableV3 == "" {
		rc.RegenerateComparableV3()
	}
}

// fixTypeNum reads rc.Type (a string) and converts it to a number, which is stored in rc.TypeNum.
func (rc *RecordConfig) fixTypeNum() {
	switch rc.Type {
	case "IGNORE":
		return
	case "IMPORT_TRANSFORM":
		return
	default:
		var err error
		tn, err := dnsutilv2.StringToType(rc.Type)
		if err != nil {
			panic(fmt.Sprintf("BUG: FixUp: Unknown type %s", rc.Type))
		}
		rc.TypeNum = tn
	}
	// fmt.Printf("DEBUG: fixTypeNum(%q) = %d\n", rc.Type, rc.TypeNum)
}

// RegenerateComparableV3 generates or regenerates the .ComparableV3 field from the current .RDATA. It does not modify .RDATA.
func (rc *RecordConfig) RegenerateComparableV3() {
	switch rc.Type {

	case "IGNORE":
		return

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

// copyLegacyFieldsToRD uses the legacy fields to generate the RDATA for the
// record. It is used to protect backwards compatibility for providers that
// have not yet been converted to RecordConfig V3.
// Always call the appropriate `Make*()` function to generate `rd`.
// This will go away when the migration to RecordConfig V3 is complete.
// For newer record types, this function will be a no-op as they have no legacy fields.
func (rc *RecordConfig) copyLegacyFieldsToRD(origin string) {

	rc.fixTypeNum()

	if rc.TypeNum == 0 {
		switch rc.Type {
		case "IGNORE":
			return
		}
		fmt.Printf("DEBUG: copyLegacyFieldsToRD: typeNum=0, Type=%q, Comparablev3=%q\n", rc.Type, rc.ComparableV3)
		return
	}

	switch rc.TypeNum {

	// These record types have no fields in RecordConfig (other than .rdata) to backfill.

	case privatetypes.TypeAKAMAICDN:
		// no-op
	case privatetypes.TypeAKAMAITLC:
		rd, err := privatetypesrdata.MakeAKAMAITLC(origin, nil, rc.AnswerType, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case privatetypes.TypeBUNNYDNSPZ:
		// no-op
	case privatetypes.TypeBUNNYDNSRDR:
		// no-op
	case privatetypes.TypeCLOUDFLAREAPISINGLEREDIRECT:
		// no-op
	case privatetypes.TypeCLOUDNSWR:
		// no-op
	case privatetypes.TypeLUA:
		// no-op
	case privatetypes.TypeNETLIFY:
		// no-op
	case privatetypes.TypeNETLIFYV6:
		// no-op
	//case "IMPORT_TRANSFORM":
	//rc.ClearRDATA()

	// These record types need to pull from their legacy fields in RecordConfig to make the RDATA.

	case dnsv2.TypeA:
		rd, err := MakeA(origin, nil, rc.GetTargetIP())
		errorChk(err)
		rc.SetRDATA(rd)
	case privatetypes.TypeALIAS:
		rd, err := privatetypesrdata.MakeALIAS(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeAAAA:
		rd, err := MakeAAAA(origin, nil, rc.GetTargetIP())
		errorChk(err)
		rc.SetRDATA(rd)
	case privatetypes.TypeADGUARDHOMEAPASSTHROUGH:
		rd, err := privatetypesrdata.MakeADGUARDHOMEAPASSTHROUGH(origin, nil)
		errorChk(err)
		rc.SetRDATA(rd)
	case privatetypes.TypeADGUARDHOMEAAAAPASSTHROUGH:
		rd, err := privatetypesrdata.MakeADGUARDHOMEAAAAPASSTHROUGH(origin, nil)
		errorChk(err)
		rc.SetRDATA(rd)
	case privatetypes.TypeAZUREALIAS:
		rd, err := privatetypesrdata.MakeAZUREALIAS(origin, nil, rc.AzureAlias["type"], rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)

	case dnsv2.TypeCAA:
		rd, err := MakeCAA(origin, nil, rc.CaaFlag, rc.CaaTag, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeCNAME:
		rd, err := MakeCNAME(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case privatetypes.TypeCFWORKERROUTE:
		part := strings.SplitN(rc.GetTargetField(), ",", 2)
		rd, err := privatetypesrdata.MakeCFWORKERROUTE(origin, nil, part[0], part[1])
		errorChk(err)
		rc.SetRDATA(rd)

	case dnsv2.TypeDHCID:
		rd, err := MakeDHCID(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeDNAME:
		rd, err := MakeDNAME(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeDNSKEY:
		rd, err := MakeDNSKEY(origin, nil, rc.DnskeyFlags, rc.DnskeyProtocol, rc.DnskeyAlgorithm, rc.DnskeyPublicKey)
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeDS:
		rd, err := MakeDS(origin, nil, rc.DsKeyTag, rc.DsAlgorithm, rc.DsDigestType, rc.DsDigest)
		errorChk(err)
		rc.SetRDATA(rd)

	case privatetypes.TypeFRAME:
		rd, err := privatetypesrdata.MakeFRAME(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)

	case dnsv2.TypeHTTPS:
		rd, err := MakeHTTPS(origin, nil, rc.SvcPriority, rc.GetTargetField(), rc.SvcParams)
		if err != nil {
			s := fmt.Sprintf("BUG: FixUp: MakeHTTPS failed for record %s IN %s %s: %v", rc.NameFQDN, rc.Type, rc.GetTargetField(), err)
			fmt.Println(s)
			panic(s)
		}
		rc.SetRDATA(rd)

	case dnsv2.TypeLOC:
		rd, err := MakeLOC(origin, nil, rc.LocVersion, rc.LocSize, rc.LocHorizPre, rc.LocVertPre, rc.LocLatitude, rc.LocLongitude, rc.LocAltitude)
		errorChk(err)
		rc.SetRDATA(rd)

	case privatetypes.TypeMIKROTIKFWD:
		rd, err := privatetypesrdata.MakeMIKROTIKFWD(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case privatetypes.TypeMIKROTIKNXDOMAIN:
		rd, err := privatetypesrdata.MakeMIKROTIKNXDOMAIN(origin, nil)
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeMX:
		rd, err := MakeMX(origin, nil, rc.MxPreference, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)

	case dnsv2.TypeNS:
		rd, err := MakeNS(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeNAPTR:
		rd, err := MakeNAPTR(origin, nil, rc.NaptrOrder, rc.NaptrPreference, rc.NaptrFlags, rc.NaptrService, rc.NaptrRegexp, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)

	case dnsv2.TypeOPENPGPKEY:
		rd, err := MakeOPENPGPKEY(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)

	case privatetypes.TypePORKBUNURLFWD:
		rd, err := privatetypesrdata.MakePORKBUNURLFWD(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)

	case dnsv2.TypePTR:
		rd, err := MakePTR(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)

	case dnsv2.TypeRP:
		//rd, err = MakeRP(origin, rc.F.(dnsv1.RP).Mbox, rc.F.(dnsv1.RP).Txt)
		// RP is native to RecordConfigV3. No FixUP is needed or possible.
	case privatetypes.TypeR53ALIAS:
		rd, err := privatetypesrdata.MakeR53ALIAS(origin, nil,
			rc.R53Alias["type"],
			rc.GetTargetField(),
			rc.R53Alias["evaluate_target_health"],
			rc.R53Alias["zone_id"],
		)
		errorChk(err)
		rc.SetRDATA(rd)

	case dnsv2.TypeSMIMEA:
		rd, err := MakeSMIMEA(origin, nil, rc.SmimeaUsage, rc.SmimeaSelector, rc.SmimeaMatchingType, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeSOA:
		rd, err := MakeSOA(origin, nil, rc.GetTargetField(), rc.SoaMbox, rc.SoaSerial, rc.SoaRefresh, rc.SoaRetry, rc.SoaExpire, rc.SoaMinttl)
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeSRV:
		rd, err := MakeSRV(origin, nil, rc.SrvPriority, rc.SrvWeight, rc.SrvPort, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeSSHFP:
		rd, err := MakeSSHFP(origin, nil, rc.SshfpAlgorithm, rc.SshfpFingerprint, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeSVCB:
		rd, err := MakeSVCB(origin, nil, rc.SvcPriority, rc.GetTargetField(), rc.SvcParams)
		errorChk(err)
		rc.SetRDATA(rd)

	case dnsv2.TypeTLSA:
		rd, err := MakeTLSA(origin, nil, rc.TlsaUsage, rc.TlsaSelector, rc.TlsaMatchingType, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case dnsv2.TypeTXT:
		rd, err := MakeTXT(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)

	case privatetypes.TypeURL:
		rd, err := privatetypesrdata.MakeURL(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)
	case privatetypes.TypeURL301:
		rd, err := privatetypesrdata.MakeURL301(origin, nil, rc.GetTargetField())
		errorChk(err)
		rc.SetRDATA(rd)

	default:
		panic(fmt.Sprintf("RDATA FIXUP NOT IMPLEMENTED TYPE=%q", rc.Type))
	}
}

func errorChk(err error) {
	if err == nil {
		return
	}
	fmt.Printf("BUG: FixUp: Make$TYPE() failed: %v\n", err)
}
