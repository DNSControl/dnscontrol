package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type MIKROTIKFWD struct {
	ForwardTo string
}

func (rd MIKROTIKFWD) Len() int {
	return len(rd.String())
}

func (rd MIKROTIKFWD) String() string {
	return txtutil.Zoneify([]string{rd.ForwardTo})
}

func MakeMIKROTIKFWD(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return MIKROTIKFWD{}, fmt.Errorf("MIKROTIK_FWD expects 1 arguments, got %d: %+v", len(args), args)
	}
	return MIKROTIKFWD{
		ForwardTo: mustbe.RawString(args[0]),
	}, nil
}
