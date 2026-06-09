package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
)

type BUNNYDNSRDR struct {
}

func (rd BUNNYDNSRDR) Len() int {
	return 0
}

func (rd BUNNYDNSRDR) String() string {
	return ""
}

func MakeBUNNYDNSRDR(origin string, args []any, _ map[string]string) (dnsv2.RDATA, error) {
	if len(args) != 0 {
		return BUNNYDNSRDR{}, fmt.Errorf("BUNNY_DNS_RDR expects 0 arguments, got %d: %+v", len(args), args)
	}
	return BUNNYDNSRDR{}, nil
}
