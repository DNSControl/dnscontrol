package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// ADGUARDHOME_AAAA_PASSTHROUGH

func init() {
	Register(TypeADGUARDHOMEAAAAPASSTHROUGH, "ADGUARDHOME_AAAA_PASSTHROUGH", func() dnsv2.RR { return new(ADGUARDHOMEAAAAPASSTHROUGH) }, privatetypesrdata.MakeADGUARDHOMEAAAAPASSTHROUGH)
}

const TypeADGUARDHOMEAAAAPASSTHROUGH = uint16(65281)

type ADGUARDHOMEAAAAPASSTHROUGH struct {
	Hdr dnsv2.Header

	privatetypesrdata.ADGUARDHOMEAAAAPASSTHROUGH
}

// Typer interface.

func (rr *ADGUARDHOMEAAAAPASSTHROUGH) Type() uint16 { return TypeADGUARDHOMEAAAAPASSTHROUGH }

// RR interface.

func (rr *ADGUARDHOMEAAAAPASSTHROUGH) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *ADGUARDHOMEAAAAPASSTHROUGH) Len() int {
	return rr.Hdr.Len()
}
func (rr *ADGUARDHOMEAAAAPASSTHROUGH) Data() dnsv2.RDATA {
	return &privatetypesrdata.ADGUARDHOMEAAAAPASSTHROUGH{}
}
func (rr *ADGUARDHOMEAAAAPASSTHROUGH) Clone() dnsv2.RR {
	return &ADGUARDHOMEAAAAPASSTHROUGH{
		rr.Hdr,
		privatetypesrdata.ADGUARDHOMEAAAAPASSTHROUGH{}}
}
func (rr *ADGUARDHOMEAAAAPASSTHROUGH) String() string {
	return rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tADGUARDHOME_AAAA_PASSTHROUGH" // RDATA is empty.
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *ADGUARDHOMEAAAAPASSTHROUGH) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 0 {
		return fmt.Errorf("ADGUARDHOME_AAAA_PASSTHROUGH requires exactly 0 arguments, got %d", len(args))
	}
	return nil
}
