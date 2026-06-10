package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// NETLIFYV6

func init() {
	Register(TypeNETLIFYV6, "NETLIFYV6", func() dnsv2.RR { return new(NETLIFYV6) }, privatetypesrdata.MakeNETLIFYV6)
}

const TypeNETLIFYV6 = uint16(65296)

type NETLIFYV6 struct {
	Hdr dnsv2.Header

	privatetypesrdata.NETLIFYV6
}

// Typer interface.

func (rr *NETLIFYV6) Type() uint16 { return TypeNETLIFYV6 }

// RR interface.

func (rr *NETLIFYV6) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *NETLIFYV6) Len() int {
	return rr.Hdr.Len()
}
func (rr *NETLIFYV6) Data() dnsv2.RDATA {
	return &privatetypesrdata.NETLIFYV6{}
}
func (rr *NETLIFYV6) Clone() dnsv2.RR {
	return &NETLIFYV6{
		rr.Hdr,
		privatetypesrdata.NETLIFYV6{}}
}
func (rr *NETLIFYV6) String() string {
	return rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tNETLIFYV6" // RDATA is empty.
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *NETLIFYV6) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 0 {
		return fmt.Errorf("NETLIFYV6 requires exactly 0 arguments, got %d", len(args))
	}
	return nil
}
