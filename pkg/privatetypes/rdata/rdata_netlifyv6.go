package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
)

type NETLIFYV6 struct {
}

func (rd NETLIFYV6) Len() int {
	return 0
}

func (rd NETLIFYV6) String() string {
	return ""
}

func MakeNETLIFYV6(origin string, args ...any) (dnsv2.RDATA, error) {
	if len(args) != 0 {
		return NETLIFYV6{}, fmt.Errorf("NETLIFYV6 expects 0 arguments, got %d: %+v", len(args), args)
	}
	return NETLIFYV6{}, nil
}
