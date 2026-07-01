package models

import (
	"bytes"
	"fmt"
	"log"
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
	rc.SetRDATA(rd)
	rc.ComparableV3 = ""
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
	rrv2, err := MyNewData(rty, contents, origin)
	if err != nil {
		return fmt.Errorf("could not parse SVCB record: %w", err)
	}
	rc.SetRDATA(rrv2)
	// rc.ComparableV3 = ""

	switch r := record.(type) {
	case *dnsv1.HTTPS:
		return rc.SetTargetSVCB(r.Priority, r.Target, r.Value)
	case *dnsv1.SVCB:
		return rc.SetTargetSVCB(r.Priority, r.Target, r.Value)
	}

	if rc.SvcPriority == 0 {
		rc.SetRDATA(&dnsrdatav2.SVCB{Priority: rc.SvcPriority, Target: rc.GetTargetField()})
	} else {
		rd, err := MyNewData(dnsv2.TypeSVCB, fmt.Sprintf("%d %s %s", rc.SvcPriority, rc.GetTargetField(), rc.SvcParams), origin)
		if err != nil {
			panic(fmt.Sprintf("BUG: Failed to create RDATA for HTTPS record: %v", err))
		}
		rc.SetRDATA(rd)
	}
	rc.FixUp("")

	return nil
}

// GetSVCBValue returns the SVCB Key/Values as a list of Key/Values.
// Used to construct dnsv.RR of type SVCB or HTTPS. (This is legacy code that should go away eventualy).
func (rc *RecordConfig) GetSVCBValue() []dnsv1.SVCBKeyValue {
	var s string
	if rc.GetRDATA() != nil {
		s = fmt.Sprintf("%s %s %s", rc.NameFQDN, rc.Type, rc.GetRDATA().String())
	} else {
		s = fmt.Sprintf("%s %s %d %s %s", rc.NameFQDN, rc.Type, rc.SvcPriority, rc.target, rc.SvcParams)
	}
	record, err := dnsv1.NewRR(s)
	if err != nil {
		log.Fatalf("could not parse SVCB record: %s", err)
	}
	switch r := record.(type) {
	case *dnsv1.HTTPS:
		return r.Value
	case *dnsv1.SVCB:
		return r.Value
	}

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

// SVCBHydrateDesiredEchIgnore finds any ECH=IGNORE parameters (stored as
// ECH=1000) and substitute the existing ECH value instead.
func SVCBHydrateDesiredEchIgnore(existing, desired Records) {
	cache := map[string][]byte{}

	// Gather the ECH= values from "existing"
	for _, rec := range existing {
		if rec.TypeNum == dnsv2.TypeSVCB || rec.TypeNum == dnsv2.TypeHTTPS {
			k := svcbEncKey(rec)
			v := svcbEncValue(rec)
			if len(v) == 0 {
				continue
			}
			cache[k] = v
		}
	}
	svcbDumpCache(cache)

	// Find any ECH=1000 values from "desired" and replace them.
	for _, rec := range desired {
		if rec.TypeNum == dnsv2.TypeSVCB || rec.TypeNum == dnsv2.TypeHTTPS {
			k := svcbEncKey(rec)
			if _, ok := cache[k]; ok {
				// if eValue, ok := cache[k]; ok {
				rd := rec.GetRDATA().(dnsrdatav2.SVCB)
				desiredPairs := rd.Value
				// fmt.Printf("DEBUG: SVCB %q exists in existing (%q) and desired (%v)\n", k, b64.StdEncoding.EncodeToString(eValue), desiredPairs)
				newPairs, found := svcbReplaceIGNOREWithData(desiredPairs, cache, rec)
				if found {
					rd.Value = newPairs
					rec.SetRDATA(rd)
					rec.ComparableV3 = ""
					rec.FixUp("")
					err := backfill(rec)
					if err != nil {
						panic(err)
					}
					// fmt.Printf("DEBUG: NEW SVCB %s\n", rec.String())
				}
			}
		}
	}
}

func svcbEncKey(rec *RecordConfig) string {
	rd := rec.GetRDATA().(dnsrdatav2.SVCB)
	//return fmt.Sprintf("%s %v %s", rec.NameFQDN, rd.Priority, rd.Target)
	return fmt.Sprintf("%s:%v", rec.NameFQDN, rd.Priority)
}

func svcbEncValue(rec *RecordConfig) []byte {
	rd := rec.GetRDATA().(dnsrdatav2.SVCB)
	for _, pair := range rd.Value {
		kNum := svcbv2.PairToKey(pair)
		if kNum != svcbv2.KeyEchConfig {
			continue
		}
		return pair.(*svcbv2.ECHCONFIG).ECH
	}
	return nil
}

func svcbDumpCache(cache map[string][]byte) {
	// if len(cache) > 0 {
	// 	fmt.Printf("\n##### SVCB Ech Cache:\n")
	// 	for k, v := range cache {
	// 		fmt.Printf("##### CACHE k=%q v=%q\n", k, base64.StdEncoding.EncodeToString(v))
	// 	}
	// 	fmt.Printf("#####\n\n")
	// }
}

// svcbReplaceIGNOREWithData searches through pairs for an ECH value == the magic number ("1000").
func svcbReplaceIGNOREWithData(pairs []svcbv2.Pair, cache map[string][]byte, rec *RecordConfig) ([]svcbv2.Pair, bool) {
	var result []svcbv2.Pair
	found := false

	for _, p := range pairs {
		// fmt.Printf("DEBUG: svcb copying %s=%v\n", svcbv2.KeyToString(svcbv2.PairToKey(p)), p)
		// pstr := fmt.Sprintf("%v", p)
		// if pstr == "1000" {
		// 	fmt.Printf("HERE\n")
		// }
		switch v := p.(type) {
		case *svcbv2.ECHCONFIG:
			//result = append(result, p)
			// fmt.Printf("DEBUG ech=%v\n", b64.StdEncoding.EncodeToString(v.ECH))
			if bytes.Equal(v.ECH, []byte{215, 77, 52}) { // "1000"
				found = true
				// fmt.Printf("DEBUG: ECH! FOUND %v\n", v)
				ech := p.(*svcbv2.ECHCONFIG)
				ech.ECH = cache[svcbEncKey(rec)]
				result = append(result, ech)

			} else {
				// fmt.Printf("DEBUG: ECH! NOT FOUND %v\n", v)
				result = append(result, p)
			}

		default:
			result = append(result, p)
		}
	}

	// Rebuild .ComparableV3 no matter what.
	rec.ComparableV3 = ""
	rec.FixUp("")

	return result, found
}
