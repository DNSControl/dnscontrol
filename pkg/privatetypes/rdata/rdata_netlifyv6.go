package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

type NETLIFYV6 struct {
}

func (rd NETLIFYV6) Len() int {
	return 0
}

func (rd NETLIFYV6) String() string {
	return ""
}

func MakeNETLIFYV6(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 0 {
		return NETLIFYV6{}, fmt.Errorf("NETLIFYV6 expects 0 arguments, got %d: %+v", len(args), args)
	}
	return NETLIFYV6{}, nil
}
