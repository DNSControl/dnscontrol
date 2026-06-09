package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

type BUNNYDNSRDR struct {
}

func (rd BUNNYDNSRDR) Len() int {
	return 0
}

func (rd BUNNYDNSRDR) String() string {
	return ""
}

func MakeBUNNYDNSRDR(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 0 {
		return BUNNYDNSRDR{}, fmt.Errorf("BUNNY_DNS_RDR expects 0 arguments, got %d: %+v", len(args), args)
	}
	return BUNNYDNSRDR{}, nil
}
