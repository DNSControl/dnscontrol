package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// MIKROTIK_FWD

func init() {
	Register(TypeMIKROTIKFWD, "MIKROTIK_FWD", func() dnsv2.RR { return new(MIKROTIKFWD) }, privatetypesrdata.MakeMIKROTIKFWD)
}

const TypeMIKROTIKFWD = uint16(65293)

type MIKROTIKFWD struct {
	Hdr dnsv2.Header

	privatetypesrdata.MIKROTIKFWD
	// ForwardTo            string
}

// Typer interface.

func (rr *MIKROTIKFWD) Type() uint16 { return TypeMIKROTIKFWD }

// RR interface.

func (rr *MIKROTIKFWD) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *MIKROTIKFWD) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *MIKROTIKFWD) Data() dnsv2.RDATA {
	return &privatetypesrdata.MIKROTIKFWD{ForwardTo: rr.ForwardTo}
}
func (rr *MIKROTIKFWD) Clone() dnsv2.RR {
	return &MIKROTIKFWD{
		Hdr: rr.Hdr,
		MIKROTIKFWD: privatetypesrdata.MIKROTIKFWD{
			ForwardTo: rr.ForwardTo,
		}}
}
func (rr *MIKROTIKFWD) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tMIKROTIK_FWD\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *MIKROTIKFWD) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 1 {
		return fmt.Errorf("MIKROTIK_FWD requires exactly 1 arguments, got %d: %v", len(args), args)
	}
	rr.ForwardTo = mustbe.RawString(args[0])
	return nil
}
