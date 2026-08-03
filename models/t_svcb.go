package models

import (
	"bytes"
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	svcbv2 "codeberg.org/miekg/dns/svcb"
)

func (rc *RecordConfig) targetCombinedSVCBRaw() string {
	if rc.SvcParams == "" {
		return fmt.Sprintf("%d %s", rc.SvcPriority, rc.target)
	}
	return fmt.Sprintf("%d %s %s", rc.SvcPriority, rc.target, rc.SvcParams)
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

// Svcbv2ValueToString converts a SVCB value list to a string.
// Typical usage: models.Svcbv2ValueToString(rc.AsHTTPS().Value)
// Does NOT generate quotes around values in key=value pairs.
func Svcbv2ValueToString(pairs []svcbv2.Pair) string {
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
	// svcbDumpCache(cache)

	// Find any ECH=1000 values from "desired" and replace them.
	for _, rec := range desired {
		if rec.TypeNum == dnsv2.TypeSVCB || rec.TypeNum == dnsv2.TypeHTTPS {
			k := svcbEncKey(rec)
			if _, ok := cache[k]; ok {
				// if eValue, ok := cache[k]; ok {
				rd := rec.AsSVCB()
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
	rd := rec.AsSVCB()
	return fmt.Sprintf("%s:%v", rec.NameFQDN, rd.Priority)
}

func svcbEncValue(rec *RecordConfig) []byte {
	rd := rec.AsSVCB()
	for _, pair := range rd.Value {
		kNum := svcbv2.PairToKey(pair)
		if kNum != svcbv2.KeyEchConfig {
			continue
		}
		return pair.(*svcbv2.ECHCONFIG).ECH
	}
	return nil
}

// func svcbDumpCache(cache map[string][]byte) {
// 	if len(cache) > 0 {
// 		fmt.Printf("\n##### SVCB Ech Cache:\n")
// 		for k, v := range cache {
// 			fmt.Printf("##### CACHE k=%q v=%q\n", k, base64.StdEncoding.EncodeToString(v))
// 		}
// 		fmt.Printf("#####\n\n")
// 	}
// }

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
