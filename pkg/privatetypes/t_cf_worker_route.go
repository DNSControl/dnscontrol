package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// CF_WORKER_ROUTE

func init() {
	Register(TypeCFWORKERROUTE, "CF_WORKER_ROUTE", func() dnsv2.RR { return new(CFWORKERROUTE) }, privatetypesrdata.MakeCFWORKERROUTE)
}

const TypeCFWORKERROUTE = uint16(65288)

type CFWORKERROUTE struct {
	Hdr dnsv2.Header

	privatetypesrdata.CFWORKERROUTE
	// When                 string
	// Then                 string
}

// Typer interface.

func (rr *CFWORKERROUTE) Type() uint16 { return TypeCFWORKERROUTE }

// RR interface.

func (rr *CFWORKERROUTE) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *CFWORKERROUTE) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *CFWORKERROUTE) Data() dnsv2.RDATA {
	return &privatetypesrdata.CFWORKERROUTE{When: rr.When, Then: rr.Then}
}
func (rr *CFWORKERROUTE) Clone() dnsv2.RR {
	return &CFWORKERROUTE{
		Hdr: rr.Hdr,
		CFWORKERROUTE: privatetypesrdata.CFWORKERROUTE{
			When: rr.When,
			Then: rr.Then,
		}}
}
func (rr *CFWORKERROUTE) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tCF_WORKER_ROUTE\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *CFWORKERROUTE) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 2 {
		return fmt.Errorf("CF_WORKER_ROUTE requires exactly 2 arguments, got %d: %v", len(args), args)
	}
	rr.When = mustbe.RawString(args[0])
	rr.Then = mustbe.RawString(args[1])
	return nil
}
