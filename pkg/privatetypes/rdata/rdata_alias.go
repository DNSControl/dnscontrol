package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type ALIAS struct {
	Target string
}

func (rd ALIAS) Len() int {
	return len(rd.String())
}

func (rd ALIAS) String() string {
	return txtutil.Zoneify([]string{rd.Target})
}

func MakeALIAS(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return ALIAS{}, fmt.Errorf("ALIAS expects 1 arguments, got %d: %+v", len(args), args)
	}
	return ALIAS{
		Target: mustbe.TargetHost(origin, args[0]),
	}, nil
}
