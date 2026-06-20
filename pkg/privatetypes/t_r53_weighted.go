package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// R53_WEIGHTED

func init() {
	Register(TypeR53WEIGHTED, "R53_WEIGHTED", func() dnsv2.RR { return new(R53WEIGHTED) }, privatetypesrdata.MakeR53WEIGHTED)
}

const TypeR53WEIGHTED = uint16(65321)

type R53WEIGHTED struct {
	Hdr dnsv2.Header

	privatetypesrdata.R53WEIGHTED
	// Weight               uint8
	// SetID                string
}

// Typer interface.

func (rr *R53WEIGHTED) Type() uint16 { return TypeR53WEIGHTED }

// RR interface.

func (rr *R53WEIGHTED) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *R53WEIGHTED) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *R53WEIGHTED) Data() dnsv2.RDATA {
	return &privatetypesrdata.R53WEIGHTED{Weight: rr.Weight, SetID: rr.SetID}
}
func (rr *R53WEIGHTED) Clone() dnsv2.RR {
	return &R53WEIGHTED{
		Hdr: rr.Hdr,
		R53WEIGHTED: privatetypesrdata.R53WEIGHTED{
			Weight: rr.Weight,
			SetID:  rr.SetID,
		}}
}
func (rr *R53WEIGHTED) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tR53_WEIGHTED\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *R53WEIGHTED) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 2 {
		return fmt.Errorf("R53_WEIGHTED requires exactly 2 arguments, got %d: %v", len(args), args)
	}
	rr.Weight = mustbe.Uint8(args[0])
	rr.SetID = mustbe.RawString(args[1])
	return nil
}
