package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type AKAMAICDN struct {
	Target               string
}

func (rd AKAMAICDN) Len() int {
	return len(rd.String())
}

func (rd AKAMAICDN) String() string {
	return txtutil.Zoneify([]string{rd.Target})
}

func MakeAKAMAICDN(origin string, args []any, _ map[string]string) (dnsv2.RDATA, error) {
	if len(args) != 1 {
		return AKAMAICDN{}, fmt.Errorf("AKAMAICDN expects 1 arguments, got %d: %+v", len(args), args)
	}
	return AKAMAICDN{
		Target: mustbe.TargetHost(origin, args[0]),
	}, nil
}
