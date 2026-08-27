package models

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

// ChangeTypeToCNAME changes rc into a CNAME pointing at target. target will be
// passed to mustbe.TargetHost() to assure it is (or convert it to) a FQDN+".",
// canonicalized to ASCII, and ToLower.
func (rc *RecordConfig) ChangeTypeToCNAME(dc *DomainConfig, target string) {
	// Change the Type/TypeNum
	rc.Type = "CNAME"
	rc.TypeNum = dnsv2.TypeCNAME

	// Store the new RDATA:
	rd, err := MakeCNAME(dc.Name, nil, nrc.Flags{}, target)
	if err != nil {
		panic(fmt.Sprintf("failed ChangeTypeToCNAME: err=%s", err)) // Should not happen.
	}
	rc.SetRDATA(rd)
}
