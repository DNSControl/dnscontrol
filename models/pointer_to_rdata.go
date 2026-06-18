package models

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
)

func AssureItsAPointer(rd dnsv2.RDATA) dnsv2.RDATA {
	if rd == nil {
		return rd
	}
	typeName := fmt.Sprintf("%T", rd)
	if strings.HasPrefix(typeName, "*rdata.") || strings.HasPrefix(typeName, "*privatetypesrdata.") {
		return rd
	}
	//fmt.Printf("DEBUG: AssureItsAPointer called with %t. Please fix the parent so I don't have to:\n%s", rd, debug.Stack())
	switch v := rd.(type) {
	case dnsrdatav2.A:
		return &v
	case dnsrdatav2.AAAA:
		return &v
	case dnsrdatav2.CNAME:
		return &v
	case dnsrdatav2.MX:
		return &v
	case dnsrdatav2.NS:
		return &v
	case dnsrdatav2.RP:
		return &v
	case dnsrdatav2.SVCB:
		return &v
	case dnsrdatav2.TXT:
		return &v
	}
	fmt.Printf("\nDEBUG: AssureItsAPointer: Please add %T\n\n", rd)
	return rd
}
