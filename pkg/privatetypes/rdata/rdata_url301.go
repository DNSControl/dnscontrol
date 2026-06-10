package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type URL301 struct {
	Location           string
	PorkbunIncludePath bool
	PorkbunWildCard    bool
}

func (rd URL301) Len() int {
	return len(rd.String())
}

func (rd URL301) String() string {
	return txtutil.Zoneify([]string{rd.Location, fmt.Sprintf("%t", rd.PorkbunIncludePath), fmt.Sprintf("%t", rd.PorkbunWildCard)})
}

func MakeURL301(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 3 {
		return URL301{}, fmt.Errorf("URL301 expects 3 arguments, got %d: %+v", len(args), args)
	}
	return URL301{
		Location:           mustbe.TargetHost(origin, args[0]),
		PorkbunIncludePath: mustbe.Bool(args[1]),
		PorkbunWildCard:    mustbe.Bool(args[2]),
	}, nil
}
