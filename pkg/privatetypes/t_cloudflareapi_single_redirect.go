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

const TypeCLOUDFLAREAPISINGLEREDIRECT = 65289

type CLOUDFLAREAPISINGLEREDIRECT struct {
	Hdr dnsv2.Header

	privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT
	// Name                 string
	// Code                 uint16
	// When                 string
	// Then                 string
}

// Typer interface.

func (rr *CLOUDFLAREAPISINGLEREDIRECT) Type() uint16 { return TypeCLOUDFLAREAPISINGLEREDIRECT }

// RR interface.

func (rr *CLOUDFLAREAPISINGLEREDIRECT) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *CLOUDFLAREAPISINGLEREDIRECT) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *CLOUDFLAREAPISINGLEREDIRECT) Data() dnsv2.RDATA {
	return &privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT{Name: rr.Name, Code: rr.Code, When: rr.When, Then: rr.Then}
}
func (rr *CLOUDFLAREAPISINGLEREDIRECT) Clone() dnsv2.RR {
	return &CLOUDFLAREAPISINGLEREDIRECT{
		Hdr: rr.Hdr,
		CLOUDFLAREAPISINGLEREDIRECT: privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT{
			Name: rr.Name,
			Code: rr.Code,
			When: rr.When,
			Then: rr.Then,
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
	rr.Name = mustbe.RawString(args[0])
	rr.Code = mustbe.Uint16(args[1])
	rr.When = mustbe.RawString(args[2])
	rr.Then = mustbe.RawString(args[3])
	return nil
}
