package models

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

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
