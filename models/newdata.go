package models

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
)

func MyNewData(typeNum uint16, contents string, origin string) (dnsv2.RDATA, error) {
	rd, err := dnsv2.NewData(typeNum, contents, origin)
	if err != nil {
		return nil, err
	}

	rd2 := AssureItsAPointer(rd)

	return rd2, nil
}

func AssureItsAPointer(rd dnsv2.RDATA) dnsv2.RDATA {

	//        Good: `*rdata.A` or `*privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	//         Bad: `rdata.A` or `privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	//  Really Bad: `**rdata.A` or `**privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	fmt.Sprintf("%T", rd)
	if (tn[0] == "*") && (tn[1] != "*") {
		return
	}

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
	fmt.Sprintf("\n\nFIXME: AssureItsAPointer: Add %T to case statement\n\n", rd)
	return rd
}
