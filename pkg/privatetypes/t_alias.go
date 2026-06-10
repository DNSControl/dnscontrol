package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// ALIAS

func init() {
	Register(TypeALIAS, "ALIAS", func() dnsv2.RR { return new(ALIAS) }, privatetypesrdata.MakeALIAS)
}

const TypeALIAS = uint16(65285)

type ALIAS struct {
	Hdr dnsv2.Header

	privatetypesrdata.ALIAS
	// Target               string
}

// Typer interface.

func (rr *ALIAS) Type() uint16 { return TypeALIAS }

// RR interface.

func (rr *ALIAS) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *ALIAS) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *ALIAS) Data() dnsv2.RDATA {
	return &privatetypesrdata.ALIAS{Target: rr.Target}
}
func (rr *ALIAS) Clone() dnsv2.RR {
	return &ALIAS{
		Hdr: rr.Hdr,
		ALIAS: privatetypesrdata.ALIAS{
			Target: rr.Target,
		}}
}
func (rr *ALIAS) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tALIAS\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *ALIAS) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 1 {
		return fmt.Errorf("ALIAS requires exactly 1 arguments, got %d: %v", len(args), args)
	}
	rr.Target = mustbe.TargetHost("", args[0])
	return nil
}
