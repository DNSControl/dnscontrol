package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

type NETLIFY struct {
}

func (rd NETLIFY) Len() int {
	return 0
}

func (rd NETLIFY) String() string {
	return ""
}

func MakeNETLIFY(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 0 {
		return NETLIFY{}, fmt.Errorf("NETLIFY expects 0 arguments, got %d: %+v", len(args), args)
	}
	return NETLIFY{}, nil
}
