package models

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	svcbv2 "codeberg.org/miekg/dns/svcb"
	dnsv1 "github.com/miekg/dns"
)

func (rc *RecordConfig) targetCombinedSVCBRaw() string {
	if rc.SvcParams == "" {
		return fmt.Sprintf("%d %s", rc.SvcPriority, rc.target)
	}
	return fmt.Sprintf("%d %s %s", rc.SvcPriority, rc.target, rc.SvcParams)
}

// SetTargetSVCB sets the SVCB fields.
func (rc *RecordConfig) SetTargetSVCB(priority uint16, target string, params []dnsv1.SVCBKeyValue) error {

	rc.SvcPriority = priority

	if err := rc.SetTarget(target); err != nil {
		return err
	}

	paramsStr := []string{}
	for _, kv := range params {
		paramsStr = append(paramsStr, fmt.Sprintf("%s=%s", kv.Key(), kv.String()))
	}
	rc.SvcParams = strings.Join(paramsStr, " ")

	if rc.Type == "" {
		rc.Type = "SVCB"
	}
	if rc.Type != "SVCB" && rc.Type != "HTTPS" {
		panic("assertion failed: SetTargetSVCB called when .Type is not SVCB or HTTPS")
	}

	switch rc.Type {
	case "HTTPS":
		rc.TypeNum = dnsv2.TypeHTTPS
	case "SVCB":
		rc.TypeNum = dnsv2.TypeSVCB
	}
	rc.Type = dnsv2.TypeToString[rc.TypeNum]

	rd, err := MakeSVCB("", nil, priority, target, params)
	if err != nil {
		return fmt.Errorf("failed to create RDATA for SVCB record: %w", err)
	}
	rc.RDATA = rd
	rc.FixUp("")

	return nil
}

// SetTargetSVCBString is like SetTargetSVCB but accepts one big string and the origin so parsing can be done using miekg/dns.
func (rc *RecordConfig) SetTargetSVCBString(origin, contents string) error {
	if rc.Type == "" {
		rc.Type = "SVCB"
	}
	record, err := dnsv1.NewRR(fmt.Sprintf("%s. %s %s", origin, rc.Type, contents))
	if err != nil {
		return fmt.Errorf("could not parse SVCB record: %w", err)
	}

	// Hack to set .RDATA without importing miekg/dns in pkg/rtypecontrol/fixlegacy.go
	var rty uint16
	switch record.(type) {
	case *dnsv1.HTTPS:
		rty = dnsv1.TypeHTTPS
	case *dnsv1.SVCB:
		rty = dnsv1.TypeSVCB
	default:
		return fmt.Errorf("unexpected record type after parsing SVCB record: %T", record)
	}
	rrv2, err := dnsv2.NewData(rty, contents, origin)
	if err != nil {
		return fmt.Errorf("could not parse SVCB record: %w", err)
	}
	rc.RDATA = rrv2

	switch r := record.(type) {
	case *dnsv1.HTTPS:
		return rc.SetTargetSVCB(r.Priority, r.Target, r.Value)
	case *dnsv1.SVCB:
		return rc.SetTargetSVCB(r.Priority, r.Target, r.Value)
	}

	if rc.SvcPriority == 0 {
		rc.RDATA = dnsrdatav2.SVCB{Priority: rc.SvcPriority, Target: rc.GetTargetField()}
	} else {
		rd, err := dnsv2.NewData(dnsv2.TypeSVCB, fmt.Sprintf("%d %s %s", rc.SvcPriority, rc.GetTargetField(), rc.SvcParams), origin)
		if err != nil {
			panic(fmt.Sprintf("BUG: Failed to create RDATA for HTTPS record: %v", err))
		}
		rc.RDATA = rd
	}
	rc.FixUp(".")

	return nil
}

// svcbv2ValueToString converts a SVCB value list to a string.
// TODO(tlim): THIS NEEDS A UNIT TEST!
func svcbv2ValueToString(pairs []svcbv2.Pair) string {
	var sb strings.Builder
	for i, p := range pairs {
		if i > 0 {
			sb.WriteString(" ")
		}
		knum := svcbv2.PairToKey(p)
		k := svcbv2.KeyToString(knum)
		fmt.Fprintf(&sb, "%s=%s", k, p.String())
		//fmt.Printf("%d %s %s\n", i, k, p.String())
	}
	return sb.String()
}

// convertSVCBv1v2 converts dnsv1's struct to dnsv2's struct. It hasn't been tested extensively.
func convertSVCBv1v2(params []dnsv1.SVCBKeyValue) ([]svcbv2.Pair, error) {
	var value []svcbv2.Pair
	for _, kvV1 := range params {
		kV1 := kvV1.Key().String()
		keyCodeV2 := svcbv2.StringToKey(kV1)
		vV1 := kvV1.String()
		if len(vV1) > 2 && vV1[0] == '"' && vV1[len(vV1)-1] == '"' {
			panic("V has quotes")
		}
		//fmt.Printf("DEBUG: convertSVCBv1v2: k=%s keyCode=%d v1=%s\n", kV1, keyCodeV2, vV1)

		pairFn := svcbv2.KeyToPair(keyCodeV2)
		if pairFn == nil {
			return nil, fmt.Errorf("failed to lookup svc key: %s", kV1)
		}
		pair := pairFn()
		if svcbv2.PairToKey(pair) != keyCodeV2 {
			return nil, fmt.Errorf("key constant is not in sync: %v", keyCodeV2)
		}
		err := svcbv2.Parse(pair, vV1, "")
		if err != nil {
			return nil, fmt.Errorf("failed to parse svc pair: %s", kV1)
		}

		vV2 := pair.String()
		if len(vV2) > 2 && vV2[0] == '"' && vV2[len(vV2)-1] == '"' {
			panic("V2 has quotes")
		}
		if vV1 != vV2 {
			panic(fmt.Sprintf("conversion from v1 to v2 is not stable: key=%s v1=%s v2=%s", kV1, vV1, vV2))
		}

		value = append(value, pair)
	}

	return value, nil
}

// func SVCBHydrateDesiredEchIgnore(existing, desired Records) {

// 	// Build the list of existing ECH values.
// 	echs := gatherEchValues(existing)

// 	// // Clone the "desired" list. Its an array of pointers, so we clone the pointers. We can replace any record we want in the "desired" list without mutating the original.
// 	// newDesired := make(Records, len(desired))
// 	// copy(newDesired, desired)

// 	// Scan desired for ech=IGNORE.  Replace any records.
// 	//recs, edits := replaceSvcbIgnores(newDesired, echs)
// 	replaceSvcbIgnores(&desired, echs)

// 	// if edits {
// 	// 	return recs
// 	// }
// 	// // No changes were made, so we can return the original "desired" list to save memory.
// 	// return desired
// }

// // gatherEchValues builds a map of FQDN to ECH values for all SVCB and HTTPS
// // records in the given set of records.  This is used to support the
// // "ech=IGNORE" feature, where we want to ignore changes in the ECH value when
// // comparing records, but still show the ECH value in the output for debugging
// // purposes.
// func gatherEchValues(recs Records) map[string]*svcbv2.ECHCONFIG {
// 	echs := map[string]*svcbv2.ECHCONFIG{}
// 	for _, rec := range recs {
// 		if rec.TypeNum == dnsv2.TypeSVCB || rec.TypeNum == dnsv2.TypeHTTPS {
// 			if value := rec.GetSVCBEchConfig(); value != nil {
// 				echs[rec.NameFQDN] = value
// 			}
// 		}
// 	}
// 	return echs
// }

// TODO: Unexport this?

// // getSVCBEchConfig returns the value of the ECH parameter. The value is a pointer to a clone.
// func (rc *RecordConfig) GetSVCBEchConfig() *svcbv2.ECHCONFIG {
// 	if rc.TypeNum != dnsv2.TypeSVCB && rc.TypeNum != dnsv2.TypeHTTPS {
// 		panic("assertion failed: GetSVCBParam called when .Type is not SVCB or HTTPS")
// 	}
// 	if rc.RDATA == nil {
// 		panic("assertion failed: SVCB/HTTPS record does not have RDATA set")
// 	}

// 	for _, param := range rc.RDATA.(dnsrdatav2.SVCB).Value {
// 		key := svcbv2.PairToKey(param)
// 		if key == svcbv2.KeyEchConfig {
// 			// p := param.(*svcbv2.ECHCONFIG)
// 			// c := p.Clone()
// 			// return c.(*svcbv2.ECHCONFIG), true
// 			return param.(*svcbv2.ECHCONFIG)
// 		}
// 	}
// 	return nil
// }

// // func replaceSvcbIgnores(records Records, echs map[string]*svcbv2.ECHCONFIG) (Records, bool) {
// func replaceSvcbIgnores(records *Records, echs map[string]*svcbv2.ECHCONFIG) {
// 	// edits := false
// 	fmt.Printf("DEBUG replaceSvcbIgnores: Called with %d desired records\n", len(*records))
// 	fmt.Printf("DEBUG replaceSvcbIgnores: echs map has %d entries\n", len(echs))
// 	for k, v := range echs {
// 		fmt.Printf("DEBUG replaceSvcbIgnores: echs[%q] = %+v\n", k, v.ECH)
// 	}

// 	for _, rec := range *records {
// 		// Not HTTPS/SVCB? skip.
// 		if rec.TypeNum != dnsv2.TypeSVCB && rec.TypeNum != dnsv2.TypeHTTPS {
// 			continue
// 		}

// 		fmt.Printf("DEBUG replaceSvcbIgnores: Processing %s record %q\n", rec.Type, rec.NameFQDN)

// 		// Try to get the ECH.  Not exist Skip.
// 		ec := rec.GetSVCBEchConfig()
// 		if ec == nil {
// 			fmt.Printf("DEBUG replaceSvcbIgnores: record %q has no ECH config\n", rec.NameFQDN)
// 			continue
// 		}

// 		// Look for ech=1000, which is our magic marker that this record wants us to substitute the actual ECH value.
// 		fmt.Printf("DEBUG replaceSvcbIgnores: record %q has ech=%+v (checking if [16 0])\n", rec.NameFQDN, ec.ECH)
// 		if !bytes.Equal(ec.ECH, []byte{16, 0}) {
// 			fmt.Printf("DEBUG replaceSvcbIgnores: record %q doesn't have magic marker, skipping\n", rec.NameFQDN)
// 			continue
// 		}

// 		if true {
// 			continue
// 		}

// 		// Replace the ECH value with the value from "existing".
// 		fmt.Printf("DEBUG replaceSvcbIgnores: record %q HAS magic marker [16 0], replacing...\n", rec.NameFQDN)
// 		nEch := echs[rec.NameFQDN]
// 		if nEch != nil {
// 			fmt.Printf("DEBUG replaceSvcbIgnores: Found ECH value in echs map: %+v\n", nEch.ECH)
// 			// Actually update the record's RDATA with the new ECH value.
// 			rec.RDATA = replaceOrAddEch(rec.RDATA.(dnsrdatav2.SVCB), nEch)
// 		} else {
// 			fmt.Printf("DEBUG replaceSvcbIgnores: WARNING: ECH value NOT found in echs map for %q\n", rec.NameFQDN)
// 		}

// 		// Fix the .ComparableV3.
// 		rec.ComparableV3 = ""
// 		rec.FixUp("")
// 		fmt.Printf("DEBUG replaceSvcbIgnores: Updated ComparableV3 for %q\n", rec.NameFQDN)

// 	}
// }
