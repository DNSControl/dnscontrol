package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// MIKROTIK_NXDOMAIN

func init() {
	Register(TypeMIKROTIKNXDOMAIN, "MIKROTIK_NXDOMAIN", func() dnsv2.RR { return new(MIKROTIKNXDOMAIN) }, privatetypesrdata.MakeMIKROTIKNXDOMAIN)
}

const TypeMIKROTIKNXDOMAIN = uint16(65294)

type MIKROTIKNXDOMAIN struct {
	Hdr dnsv2.Header

	privatetypesrdata.MIKROTIKNXDOMAIN
}

// Typer interface.

func (rr *MIKROTIKNXDOMAIN) Type() uint16 { return TypeMIKROTIKNXDOMAIN }

// RR interface.

func (rr *MIKROTIKNXDOMAIN) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *MIKROTIKNXDOMAIN) Len() int {
	return rr.Hdr.Len()
}
func (rr *MIKROTIKNXDOMAIN) Data() dnsv2.RDATA {
	return &privatetypesrdata.MIKROTIKNXDOMAIN{}
}
func (rr *MIKROTIKNXDOMAIN) Clone() dnsv2.RR {
	return &MIKROTIKNXDOMAIN{
		rr.Hdr,
		privatetypesrdata.MIKROTIKNXDOMAIN{}}
}
func (rr *MIKROTIKNXDOMAIN) String() string {
	return rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tMIKROTIK_NXDOMAIN" // RDATA is empty.
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *MIKROTIKNXDOMAIN) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 0 {
		return fmt.Errorf("MIKROTIK_NXDOMAIN requires exactly 0 arguments, got %d", len(args))
	}
	return nil
}
