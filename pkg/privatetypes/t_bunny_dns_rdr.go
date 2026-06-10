package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// BUNNY_DNS_RDR

func init() {
	Register(TypeBUNNYDNSRDR, "BUNNY_DNS_RDR", func() dnsv2.RR { return new(BUNNYDNSRDR) }, privatetypesrdata.MakeBUNNYDNSRDR)
}

const TypeBUNNYDNSRDR = uint16(65320)

type BUNNYDNSRDR struct {
	Hdr dnsv2.Header

	privatetypesrdata.BUNNYDNSRDR
}

// Typer interface.

func (rr *BUNNYDNSRDR) Type() uint16 { return TypeBUNNYDNSRDR }

// RR interface.

func (rr *BUNNYDNSRDR) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *BUNNYDNSRDR) Len() int {
	return rr.Hdr.Len()
}
func (rr *BUNNYDNSRDR) Data() dnsv2.RDATA {
	return &privatetypesrdata.BUNNYDNSRDR{}
}
func (rr *BUNNYDNSRDR) Clone() dnsv2.RR {
	return &BUNNYDNSRDR{
		rr.Hdr,
		privatetypesrdata.BUNNYDNSRDR{}}
}
func (rr *BUNNYDNSRDR) String() string {
	return rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tBUNNY_DNS_RDR" // RDATA is empty.
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *BUNNYDNSRDR) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 0 {
		return fmt.Errorf("BUNNY_DNS_RDR requires exactly 0 arguments, got %d", len(args))
	}
	return nil
}
