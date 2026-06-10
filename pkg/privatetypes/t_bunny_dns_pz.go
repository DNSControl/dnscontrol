package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// BUNNY_DNS_PZ

func init() {
	Register(TypeBUNNYDNSPZ, "BUNNY_DNS_PZ", func() dnsv2.RR { return new(BUNNYDNSPZ) }, privatetypesrdata.MakeBUNNYDNSPZ)
}

const TypeBUNNYDNSPZ = uint16(65287)

type BUNNYDNSPZ struct {
	Hdr dnsv2.Header

	privatetypesrdata.BUNNYDNSPZ
}

// Typer interface.

func (rr *BUNNYDNSPZ) Type() uint16 { return TypeBUNNYDNSPZ }

// RR interface.

func (rr *BUNNYDNSPZ) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *BUNNYDNSPZ) Len() int {
	return rr.Hdr.Len()
}
func (rr *BUNNYDNSPZ) Data() dnsv2.RDATA {
	return &privatetypesrdata.BUNNYDNSPZ{}
}
func (rr *BUNNYDNSPZ) Clone() dnsv2.RR {
	return &BUNNYDNSPZ{
		rr.Hdr,
		privatetypesrdata.BUNNYDNSPZ{}}
}
func (rr *BUNNYDNSPZ) String() string {
	return rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tBUNNY_DNS_PZ" // RDATA is empty.
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *BUNNYDNSPZ) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 0 {
		return fmt.Errorf("BUNNY_DNS_PZ requires exactly 0 arguments, got %d", len(args))
	}
	return nil
}
