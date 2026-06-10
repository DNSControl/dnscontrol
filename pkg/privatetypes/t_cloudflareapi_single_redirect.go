package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// CLOUDFLAREAPI_SINGLE_REDIRECT

func init() {
	Register(TypeCLOUDFLAREAPISINGLEREDIRECT, "CLOUDFLAREAPI_SINGLE_REDIRECT", func() dnsv2.RR { return new(CLOUDFLAREAPISINGLEREDIRECT) }, privatetypesrdata.MakeCLOUDFLAREAPISINGLEREDIRECT)
}

const TypeCLOUDFLAREAPISINGLEREDIRECT = uint16(65289)

type CLOUDFLAREAPISINGLEREDIRECT struct {
	Hdr dnsv2.Header

	privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT
	// SRName               string
	// Code                 uint16
	// SRWhen               string
	// SRThen               string
}

// Typer interface.

func (rr *CLOUDFLAREAPISINGLEREDIRECT) Type() uint16 { return TypeCLOUDFLAREAPISINGLEREDIRECT }

// RR interface.

func (rr *CLOUDFLAREAPISINGLEREDIRECT) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *CLOUDFLAREAPISINGLEREDIRECT) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *CLOUDFLAREAPISINGLEREDIRECT) Data() dnsv2.RDATA {
	return &privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT{SRName: rr.SRName, Code: rr.Code, SRWhen: rr.SRWhen, SRThen: rr.SRThen}
}
func (rr *CLOUDFLAREAPISINGLEREDIRECT) Clone() dnsv2.RR {
	return &CLOUDFLAREAPISINGLEREDIRECT{
		Hdr: rr.Hdr,
		CLOUDFLAREAPISINGLEREDIRECT: privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT{
			SRName: rr.SRName,
			Code:   rr.Code,
			SRWhen: rr.SRWhen,
			SRThen: rr.SRThen,
		}}
}
func (rr *CLOUDFLAREAPISINGLEREDIRECT) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tCLOUDFLAREAPI_SINGLE_REDIRECT\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *CLOUDFLAREAPISINGLEREDIRECT) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 4 {
		return fmt.Errorf("CLOUDFLAREAPI_SINGLE_REDIRECT requires exactly 4 arguments, got %d: %v", len(args), args)
	}
	rr.SRName = mustbe.RawString(args[0])
	rr.Code = mustbe.Uint16(args[1])
	rr.SRWhen = mustbe.RawString(args[2])
	rr.SRThen = mustbe.RawString(args[3])
	return nil
}
