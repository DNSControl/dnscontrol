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

// // SetTargetSVCB sets the SVCB fields.
// Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) SetTargetSVCB(priority uint16, target string, params []dnsv1.SVCBKeyValue) error {
// 	return legacySetTargetArgs(rc, dnsv2.TypeSVCB, priority, target, params)
// }

// SetTargetSVCBString is like SetTargetSVCB but accepts one big string and the origin so parsing can be done using miekg/dns.
// Deprecated. Use models.NewRecordConfigParse() instead.
func (rc *RecordConfig) SetTargetSVCBString(origin, contents string) error {
	return legacySetTargetParse(rc, dnsv2.TypeSVCB, contents)
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

// stringToSvcbv2Values converts a string to a SVCB value list.
// TODO(tlim): THIS NEEDS A UNIT TEST!
func stringToSvcbv2Values(origin string, contents string) ([]svcbv2.Pair, error) {
	fields := strings.Fields(contents)

	var result []svcbv2.Pair

	for _, field := range fields {
		keyValue := strings.SplitN(field, "=", 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("invalid svcb.Pair: %q", field)
		}

		// Make the pair.
		pairFn := svcbv2.KeyToPair(svcbv2.StringToKey(keyValue[0]))
		pair := pairFn()

		// Strip the value of any quotes:
		v := keyValue[1]
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' { // Strip quotes if present
			v = v[1 : len(v)-1]
		}

		// Parse it:
		err := svcbv2.Parse(pair, v, origin)
		if err != nil {
			return nil, err
		}

		result = append(result, pair)
	}
	return result, nil
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
				newPairs, found := svcbReplaceIGNOREWithData(desiredPairs, cache, rec)
				if found {
					rd.Value = newPairs
					rec.SetRDATA(rd)
				}
			}
		}
	}
}

func svcbEncKey(rec *RecordConfig) string {
	rd := rec.GetRDATA().(dnsrdatav2.SVCB)
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
		switch v := p.(type) {
		case *svcbv2.ECHCONFIG:
			if bytes.Equal(v.ECH, []byte{215, 77, 52}) { // "1000"
				found = true
				ech := p.(*svcbv2.ECHCONFIG)
				ech.ECH = cache[svcbEncKey(rec)]
				result = append(result, ech)
			} else {
				result = append(result, p)
			}

		default:
			result = append(result, p)
		}
	}

	rec.RegenerateComparableV3()

	return result, found
}
