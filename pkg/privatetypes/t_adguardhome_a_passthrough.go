package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// ADGUARDHOME_A_PASSTHROUGH

func init() {
	Register(TypeADGUARDHOMEAPASSTHROUGH, "ADGUARDHOME_A_PASSTHROUGH", func() dnsv2.RR { return new(ADGUARDHOMEAPASSTHROUGH) }, privatetypesrdata.MakeADGUARDHOMEAPASSTHROUGH)
}

const TypeADGUARDHOMEAPASSTHROUGH = uint16(65280)

type ADGUARDHOMEAPASSTHROUGH struct {
	Hdr dnsv2.Header

	privatetypesrdata.ADGUARDHOMEAPASSTHROUGH
}

// Typer interface.

func (rr *ADGUARDHOMEAPASSTHROUGH) Type() uint16 { return TypeADGUARDHOMEAPASSTHROUGH }

// RR interface.

func (rr *ADGUARDHOMEAPASSTHROUGH) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *ADGUARDHOMEAPASSTHROUGH) Len() int {
	return rr.Hdr.Len()
}
func (rr *ADGUARDHOMEAPASSTHROUGH) Data() dnsv2.RDATA {
	return &privatetypesrdata.ADGUARDHOMEAPASSTHROUGH{}
}
func (rr *ADGUARDHOMEAPASSTHROUGH) Clone() dnsv2.RR {
	return &ADGUARDHOMEAPASSTHROUGH{
		rr.Hdr,
		privatetypesrdata.ADGUARDHOMEAPASSTHROUGH{}}
}
func (rr *ADGUARDHOMEAPASSTHROUGH) String() string {
	return rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tADGUARDHOME_A_PASSTHROUGH" // RDATA is empty.
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *ADGUARDHOMEAPASSTHROUGH) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 0 {
		return fmt.Errorf("ADGUARDHOME_A_PASSTHROUGH requires exactly 0 arguments, got %d", len(args))
	}
	return nil
}
