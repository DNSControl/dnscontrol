package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// FRAME

func init() {
	Register(TypeFRAME, "FRAME", func() dnsv2.RR { return new(FRAME) }, privatetypesrdata.MakeFRAME)
}

const TypeFRAME = uint16(65291)

type FRAME struct {
	Hdr dnsv2.Header

	privatetypesrdata.FRAME
	// Target               string
}

// Typer interface.

func (rr *FRAME) Type() uint16 { return TypeFRAME }

// RR interface.

func (rr *FRAME) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *FRAME) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *FRAME) Data() dnsv2.RDATA {
	return &privatetypesrdata.FRAME{Target: rr.Target}
}
func (rr *FRAME) Clone() dnsv2.RR {
	return &FRAME{
		Hdr: rr.Hdr,
		FRAME: privatetypesrdata.FRAME{
			Target: rr.Target,
		}}
}
func (rr *FRAME) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tFRAME\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *FRAME) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 1 {
		return fmt.Errorf("FRAME requires exactly 1 arguments, got %d: %v", len(args), args)
	}
	rr.Target = mustbe.RawString(args[0])
	return nil
}
