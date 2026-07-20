package prettyzone

// Generate zonefiles.
// This generates a zonefile that prioritizes beauty over efficiency.

import (
	"bytes"
	"log"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// ZoneGenData is the configuration description for the zone generator.
type ZoneGenData struct {
	Origin     string
	DefaultTTL uint32
	Records    models.Records
	Comments   []string
}

func (z *ZoneGenData) Len() int      { return len(z.Records) }
func (z *ZoneGenData) Swap(i, j int) { z.Records[i], z.Records[j] = z.Records[j], z.Records[i] }
func (z *ZoneGenData) Less(i, j int) bool {
	a, b := z.Records[i], z.Records[j]

	// Sort by name.

	//fmt.Printf("DEBUG: LabelLess(%q, %q) = %v %q %q\n", compA, compB, LabelLess(compA, compB), a.Name, b.Name)
	compA, compB := a.NameFQDN, b.NameFQDN
	// Unify FQDNs to "@". LabelLess needs FQDNs to be "@" to work properly.
	if a.Name == "@" {
		compA = "@"
	}
	if b.Name == "@" {
		compB = "@"
	}
	if compA != compB {
		return LabelLess(compA, compB)
	}

	// sub-sort by type
	if a.Type != b.Type {
		return zoneRrtypeLess(a.Type, b.Type)
	}

	// sub-sort within type:
	switch a.Type { // #rtype_variations
	case "A":
		ta2, tb2 := a.GetTargetIP(), b.GetTargetIP()
		if !ta2.Is4() || !tb2.Is4() {
			log.Fatalf("should not happen: Invalid IPv4 address: %s %s",
				a.GetTargetIP().String(), b.GetTargetIP().String())
		}
		return bytes.Compare(ta2.AsSlice(), tb2.AsSlice()) == -1
	case "AAAA":
		ta2, tb2 := a.GetTargetIP(), b.GetTargetIP()
		if !ta2.Is6() || !tb2.Is6() {
			log.Fatalf("should not happen: Invalid IPv6 address: %s %s",
				a.GetTargetIP().String(), b.GetTargetIP().String())
		}
		return ta2.Compare(tb2) == -1
	case "MX":
		// sort by priority. If they are equal, sort by Mx.
		if a.MxPreference == b.MxPreference {
			return a.GetTargetField() < b.GetTargetField()
		}
		return a.MxPreference < b.MxPreference
	case "SRV":
		// ta2, tb2 := a.(*dns.SRV), b.(*dns.SRV)
		pa, pb := a.SrvPort, b.SrvPort
		if pa != pb {
			return pa < pb
		}
		pa, pb = a.SrvPriority, b.SrvPriority
		if pa != pb {
			return pa < pb
		}
		pa, pb = a.SrvWeight, b.SrvWeight
		if pa != pb {
			return pa < pb
		}
	case "SVCB", "HTTPS":
		// sort by priority. If they are equal, sort by record.
		if a.SvcPriority == b.SvcPriority {
			return a.GetTargetField() < b.GetTargetField()
		}
		return a.SvcPriority < b.SvcPriority
	case "PTR":
		// ta2, tb2 := a.(*dns.PTR), b.(*dns.PTR)
		pa, pb := a.GetTargetField(), b.GetTargetField()
		if pa != pb {
			return pa < pb
		}
	case "CAA":
		// ta2, tb2 := a.(*dns.CAA), b.(*dns.CAA)
		// sort by tag
		pa, pb := a.CaaTag, b.CaaTag
		if pa != pb {
			return pa < pb
		}
		// then flag
		fa, fb := a.CaaFlag, b.CaaFlag
		if fa != fb {
			// flag set goes before ones without flag set
			return fa > fb
		}
	case "DS":
		pa, pb := a.DsKeyTag, b.DsKeyTag
		if pa != pb {
			return pa < pb
		}
	case "DNSKEY":
		pa, pb := a.DnskeyFlags, b.DnskeyFlags
		if pa != pb {
			return pa < pb
		}
		fa, fb := a.DnskeyProtocol, b.DnskeyProtocol
		if fa != fb {
			return fa < fb
		}
	default:
		// pass through. String comparison is sufficient.
	}
	// fmt.Printf("DEBUG: Less %q < %q == %v\n", a.String(), b.String(), a.String() < b.String())
	return a.String() < b.String()
}

// LabelLess provides a "Less" function for two labels as needed for sorting. It
// sorts labels in prefix order, to make output pretty.
func LabelLess(a, b string) bool {
	// Compare two zone labels for the purpose of sorting the RRs in a Zone.

	// If they are equal, we are done. The remainingi code can assume a != b.
	if a == b {
		return false
	}

	// Sort @ at the top, then *, then everything else lexigraphically.
	// i.e. @ always is less. * is less than everything but @.
	if a == "@" {
		return true
	}
	if b == "@" {
		return false
	}
	if a == "*" {
		return true
	}
	if b == "*" {
		return false
	}

	// Split into elements and match up last elements to first. Compare the
	// first non-equal elements.

	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	ia := len(as) - 1
	ib := len(bs) - 1

	var minIdx int
	if ia < ib {
		minIdx = len(as) - 1
	} else {
		minIdx = len(bs) - 1
	}

	// Skip the matching highest elements, then compare the next item.
	for i, j := ia, ib; minIdx >= 0; i, j, minIdx = i-1, j-1, minIdx-1 {
		// Compare as[i] < bs[j]
		// Sort @ at the top, then *, then everything else.
		// i.e. @ always is less. * is less than everything but @.
		// If both are numeric, compare as integers, otherwise as strings.

		if as[i] != bs[j] {
			// If the first element is *, it is always less.
			if i == 0 && as[i] == "*" {
				return true
			}
			if j == 0 && bs[j] == "*" {
				return false
			}

			// If the elements are both numeric, compare as integers:
			au, aerr := strconv.ParseUint(as[i], 10, 64)
			bu, berr := strconv.ParseUint(bs[j], 10, 64)
			if aerr == nil && berr == nil {
				return au < bu
			}
			// otherwise, compare as strings:
			return as[i] < bs[j]
		}
	}
	// The min top elements were equal, so the shorter name is less.
	return ia < ib
}

func zoneRrtypeLess(a, b string) bool {
	// Compare two RR types for the purpose of sorting the RRs in a Zone.

	if a == b {
		return false
	}

	// List SOAs, NSs, etc. then all others alphabetically.

	for _, t := range []string{
		"SOA", "NS", "CNAME",
		"A", "AAAA", "MX", "SRV", "TXT", "LUA",
	} {
		if a == t {
			return true
		}
		if b == t {
			return false
		}
	}
	return a < b
}
