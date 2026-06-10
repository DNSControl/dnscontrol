package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// CLOUDNS_WR

func init() {
	Register(TypeCLOUDNSWR, "CLOUDNS_WR", func() dnsv2.RR { return new(CLOUDNSWR) }, privatetypesrdata.MakeCLOUDNSWR)
}

const TypeCLOUDNSWR = uint16(65290)

type CLOUDNSWR struct {
	Hdr dnsv2.Header

	privatetypesrdata.CLOUDNSWR
	// Target               string
}

// Typer interface.

func (rr *CLOUDNSWR) Type() uint16 { return TypeCLOUDNSWR }

// RR interface.

func (rr *CLOUDNSWR) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *CLOUDNSWR) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *CLOUDNSWR) Data() dnsv2.RDATA {
	return &privatetypesrdata.CLOUDNSWR{Target: rr.Target}
}
func (rr *CLOUDNSWR) Clone() dnsv2.RR {
	return &CLOUDNSWR{
		Hdr: rr.Hdr,
		CLOUDNSWR: privatetypesrdata.CLOUDNSWR{
			Target: rr.Target,
		}}
}
func (rr *CLOUDNSWR) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tCLOUDNS_WR\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *CLOUDNSWR) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 1 {
		return fmt.Errorf("CLOUDNS_WR requires exactly 1 arguments, got %d: %v", len(args), args)
	}
	rr.Target = mustbe.RawString(args[0])
	return nil
}
