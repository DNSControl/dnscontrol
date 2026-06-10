package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// AKAMAICDN

func init() {
	Register(TypeAKAMAICDN, "AKAMAICDN", func() dnsv2.RR { return new(AKAMAICDN) }, privatetypesrdata.MakeAKAMAICDN)
}

const TypeAKAMAICDN = uint16(65282)

type AKAMAICDN struct {
	Hdr dnsv2.Header

	privatetypesrdata.AKAMAICDN
	// Target               string
}

// Typer interface.

func (rr *AKAMAICDN) Type() uint16 { return TypeAKAMAICDN }

// RR interface.

func (rr *AKAMAICDN) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *AKAMAICDN) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *AKAMAICDN) Data() dnsv2.RDATA {
	return &privatetypesrdata.AKAMAICDN{Target: rr.Target}
}
func (rr *AKAMAICDN) Clone() dnsv2.RR {
	return &AKAMAICDN{
		Hdr: rr.Hdr,
		AKAMAICDN: privatetypesrdata.AKAMAICDN{
			Target: rr.Target,
		}}
}
func (rr *AKAMAICDN) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tAKAMAICDN\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *AKAMAICDN) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 1 {
		return fmt.Errorf("AKAMAICDN requires exactly 1 arguments, got %d: %v", len(args), args)
	}
	rr.Target = mustbe.TargetHost("", args[0])
	return nil
}
