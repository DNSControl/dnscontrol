package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// AKAMAITLC

func init() {
	Register(TypeAKAMAITLC, "AKAMAITLC", func() dnsv2.RR { return new(AKAMAITLC) }, privatetypesrdata.MakeAKAMAITLC)
}

const TypeAKAMAITLC = uint16(65284)

type AKAMAITLC struct {
	Hdr dnsv2.Header

	privatetypesrdata.AKAMAITLC
	// AnswerType           string
	// Target               string
}

// Typer interface.

func (rr *AKAMAITLC) Type() uint16 { return TypeAKAMAITLC }

// RR interface.

func (rr *AKAMAITLC) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *AKAMAITLC) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *AKAMAITLC) Data() dnsv2.RDATA {
	return &privatetypesrdata.AKAMAITLC{AnswerType: rr.AnswerType, Target: rr.Target}
}
func (rr *AKAMAITLC) Clone() dnsv2.RR {
	return &AKAMAITLC{
		Hdr: rr.Hdr,
		AKAMAITLC: privatetypesrdata.AKAMAITLC{
			AnswerType: rr.AnswerType,
			Target: rr.Target,
		}}
}
func (rr *AKAMAITLC) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tAKAMAITLC\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *AKAMAITLC) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 2 {
		return fmt.Errorf("AKAMAITLC requires exactly 2 arguments, got %d: %v", len(args), args)
	}
	rr.AnswerType = mustbe.RawString(args[0])
	rr.Target = mustbe.TargetHost("", args[1])
	return nil
}
