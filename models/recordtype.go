package models

import (
	"fmt"

	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
)

// ChangeType converts rc to an rc of type newType.  This is only needed when
// converting from one type to another. Do not use this when initializing a new
// record.
//
// Typically this is used to convert an ALIAS to a CNAME, or SPF to TXT. Using
// this function future-proofs the code since eventually such changes will
// require extra steps.
func (rc *RecordConfig) ChangeType(newType string, _ string) {

	// Change the Type/TypeNum
	rc.Type = newType
	tn, err := dnsutilv2.StringToType(rc.Type)
	if err != nil {
		panic(fmt.Sprintf("BUG: ChangeType: Unknown type %s", rc.Type))
	}
	rc.TypeNum = tn

	// Clear out anything that will need to be fixed.
	rc.rdata = nil
	rc.ComparableV3 = ""
}
