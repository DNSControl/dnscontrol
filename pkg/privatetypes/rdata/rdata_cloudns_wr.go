package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type CLOUDNSWR struct {
	Target string
}

func (rd CLOUDNSWR) Len() int {
	return len(rd.String())
}

func (rd CLOUDNSWR) String() string {
	return txtutil.Zoneify([]string{rd.Target})
}

func MakeCLOUDNSWR(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return CLOUDNSWR{}, fmt.Errorf("CLOUDNS_WR expects 1 arguments, got %d: %+v", len(args), args)
	}
	return CLOUDNSWR{
		Target: mustbe.RawString(args[0]),
	}, nil
}
