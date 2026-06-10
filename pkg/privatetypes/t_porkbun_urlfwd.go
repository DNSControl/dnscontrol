package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// PORKBUN_URLFWD

func init() {
	Register(TypePORKBUNURLFWD, "PORKBUN_URLFWD", func() dnsv2.RR { return new(PORKBUNURLFWD) }, privatetypesrdata.MakePORKBUNURLFWD)
}

const TypePORKBUNURLFWD = uint16(65297)

type PORKBUNURLFWD struct {
	Hdr dnsv2.Header

	privatetypesrdata.PORKBUNURLFWD
	// Target               string
	// TypeName             string
	// IncludePath          string
	// Wildcard             string
}

// Typer interface.

func (rr *PORKBUNURLFWD) Type() uint16 { return TypePORKBUNURLFWD }

// RR interface.

func (rr *PORKBUNURLFWD) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *PORKBUNURLFWD) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *PORKBUNURLFWD) Data() dnsv2.RDATA {
	return &privatetypesrdata.PORKBUNURLFWD{Target: rr.Target, TypeName: rr.TypeName, IncludePath: rr.IncludePath, Wildcard: rr.Wildcard}
}
func (rr *PORKBUNURLFWD) Clone() dnsv2.RR {
	return &PORKBUNURLFWD{
		Hdr: rr.Hdr,
		PORKBUNURLFWD: privatetypesrdata.PORKBUNURLFWD{
			Target:      rr.Target,
			TypeName:    rr.TypeName,
			IncludePath: rr.IncludePath,
			Wildcard:    rr.Wildcard,
		}}
}
func (rr *PORKBUNURLFWD) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tPORKBUN_URLFWD\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *PORKBUNURLFWD) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 4 {
		return fmt.Errorf("PORKBUN_URLFWD requires exactly 4 arguments, got %d: %v", len(args), args)
	}
	rr.Target = mustbe.RawString(args[0])
	rr.TypeName = mustbe.RawString(args[1])
	rr.IncludePath = mustbe.RawString(args[2])
	rr.Wildcard = mustbe.RawString(args[3])
	return nil
}
