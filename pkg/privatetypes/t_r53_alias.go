package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// R53_ALIAS

func init() {
	Register(TypeR53ALIAS, "R53_ALIAS", func() dnsv2.RR { return new(R53ALIAS) }, privatetypesrdata.MakeR53ALIAS)
}

const TypeR53ALIAS = uint16(65298)

type R53ALIAS struct {
	Hdr dnsv2.Header

	privatetypesrdata.R53ALIAS
	// AliasType            string
	// Target               string
	// EvalTargetHealth     string
	// ZoneID               string
}

// Typer interface.

func (rr *R53ALIAS) Type() uint16 { return TypeR53ALIAS }

// RR interface.

func (rr *R53ALIAS) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *R53ALIAS) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *R53ALIAS) Data() dnsv2.RDATA {
	return &privatetypesrdata.R53ALIAS{AliasType: rr.AliasType, Target: rr.Target, EvalTargetHealth: rr.EvalTargetHealth, ZoneID: rr.ZoneID}
}
func (rr *R53ALIAS) Clone() dnsv2.RR {
	return &R53ALIAS{
		Hdr: rr.Hdr,
		R53ALIAS: privatetypesrdata.R53ALIAS{
			AliasType:        rr.AliasType,
			Target:           rr.Target,
			EvalTargetHealth: rr.EvalTargetHealth,
			ZoneID:           rr.ZoneID,
		}}
}
func (rr *R53ALIAS) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tR53_ALIAS\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *R53ALIAS) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 4 {
		return fmt.Errorf("R53_ALIAS requires exactly 4 arguments, got %d: %v", len(args), args)
	}
	rr.AliasType = mustbe.RawString(args[0])
	rr.Target = mustbe.TargetHost("", args[1])
	rr.EvalTargetHealth = mustbe.RawString(args[2])
	rr.ZoneID = mustbe.RawString(args[3])
	return nil
}
