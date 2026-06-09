package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
)

type BUNNYDNSPZ struct {
}

func (rd BUNNYDNSPZ) Len() int {
	return 0
}

func (rd BUNNYDNSPZ) String() string {
	return ""
}

func MakeBUNNYDNSPZ(origin string, args []any, _ map[string]string) (dnsv2.RDATA, error) {
	if len(args) != 0 {
		return BUNNYDNSPZ{}, fmt.Errorf("BUNNY_DNS_PZ expects 0 arguments, got %d: %+v", len(args), args)
	}
	return BUNNYDNSPZ{}, nil
}
