package models

import (
	"fmt"
	"runtime/debug"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
)

// SetRDATA is a setter for RecordConfig.rdata.
func (rc *RecordConfig) SetRDATA(rd dnsv2.RDATA) {
	rc.rdata = AssureItsAPointer(rd)
	rc.ValidateRDATA()
}

// GetRDATA is a getter for RecordConfig.rdata.
func (rc *RecordConfig) GetRDATA() (rd dnsv2.RDATA) {
	return rc.rdata
}

// ClearRDATA sets rc.rdata to nil. This is a workaround and will eventually be eliminated.
func (rc *RecordConfig) ClearRDATA() {
	rc.rdata = nil
}

// ValidateRDATA is used to verify that .rdata didn't accidentally get set to
// rdata (instead of *rdata).  This shouldn't be needed, but it catches coding
// mistakes.  Eventually this may become a no-op.
func (rc *RecordConfig) ValidateRDATA() {

	if rc.GetRDATA() == nil {
		return
	}

	tn := fmt.Sprintf("%T", rc.GetRDATA())
	if strings.HasPrefix(tn, "*rdata.") {
		return
	}
	if strings.HasPrefix(tn, "*privatetypesrdata.") {
		return
	}

	l := fmt.Sprintf("\nDEBUG: ValidateRDATA: %s\n", tn)
	fmt.Println(l)
	fmt.Println(string(debug.Stack()))
	// panic(l)
}

func MyNewData(typeNum uint16, contents string, origin string) (dnsv2.RDATA, error) {
	rd, err := dnsv2.NewData(typeNum, contents, origin)
	if err != nil {
		return nil, err
	}

	rd2 := assurePointerRDATA(rd)

	return rd2, nil
}

func assurePointerRDATA(rd dnsv2.RDATA) dnsv2.RDATA {

	//        Good: `*rdata.A` or `*privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	//         Bad: `rdata.A` or `privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	//  Really Bad: `**rdata.A` or `**privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	tn := fmt.Sprintf("%T", rd)
	if (tn[0] == '*') && (tn[1] != '*') {
		return rd
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
	fmt.Sprintf("\n\nFIXME: assurePointerRDATA: Add %T to case statement\n\n", rd)
	return rd
}
