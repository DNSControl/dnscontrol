package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// PORKBUN_URLFWD

func init() {
	Register(TypePORKBUNURLFWD, "PORKBUN_URLFWD", func() dnsv2.RR { return new(PORKBUNURLFWD) }, privatetypesrdata.MakePORKBUNURLFWD)
}

const TypePORKBUNURLFWD = 65297

type PORKBUNURLFWD struct {
	Hdr dnsv2.Header

	privatetypesrdata.PORKBUNURLFWD
}

// Typer interface.

func (rr *PORKBUNURLFWD) Type() uint16 { return TypePORKBUNURLFWD }

// RR interface.

func (rr *PORKBUNURLFWD) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *PORKBUNURLFWD) Len() int {
	return rr.Hdr.Len()
}
func (rr *PORKBUNURLFWD) Data() dnsv2.RDATA {
	return &privatetypesrdata.PORKBUNURLFWD{}
}
func (rr *PORKBUNURLFWD) Clone() dnsv2.RR {
	return &PORKBUNURLFWD{
		rr.Hdr,
		privatetypesrdata.PORKBUNURLFWD{}}
}
func (rr *PORKBUNURLFWD) String() string {
	return rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tPORKBUN_URLFWD" // RDATA is empty.
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *PORKBUNURLFWD) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 0 {
		return fmt.Errorf("PORKBUN_URLFWD requires exactly 0 arguments, got %d", len(args))
	}
	return nil
}
