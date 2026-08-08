package models

import (
	"fmt"
	"strings"

	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
)

// RecomputeV3Fields re-derives the cached "V3 Fields" (.RDATA and
// .ComparableV3) after a record's underlying fields have been mutated after the
// record was first constructed (for example, a provider filling in a default
// value such as an R53_ALIAS zone_id). FixUp only populates these fields when
// they are empty, so they must be cleared first; otherwise the diff engine
// (which compares .ComparableV3) would keep seeing the pre-mutation values and
// report a spurious change.
func (rc *RecordConfig) RecomputeV3Fields(origin string) {
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
}

// RegenerateComparableV3 generates or regenerates the .ComparableV3 field from the current .RDATA. It does not modify .RDATA.
func (rc *RecordConfig) RegenerateComparableV3() {
	switch rc.Type {

	case "IGNORE":
		return

	case "IMPORT_TRANSFORM":
		return

	case "SOA":
		// The comparable string for SOA intentionally excludes the serial
		// number, because the serial number changes on every update and
		// would prevent correct diffing. List it as "X" so-as it stands out
		// in debug output that the serial is intentionally excluded.
		rd := rc.AsSOA()
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
func (rc *RecordConfig) copyLegacyFieldsToRD(_ string) {

	rc.fixTypeNum()

	if rc.TypeNum == 0 {
		switch rc.Type {
		case "IGNORE":
			return
			// case "IMPORT_TRANSFORM":
			// 	return
		}
		panic(fmt.Sprintf("DEBUG: copyLegacyFieldsToRD: typeNum=0, Type=%q, Comparablev3=%q\n", rc.Type, rc.ComparableV3))
		// return
	}

}
