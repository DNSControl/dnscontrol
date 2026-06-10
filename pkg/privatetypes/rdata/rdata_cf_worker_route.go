package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type CFWORKERROUTE struct {
	When string
	Then string
}

func (rd CFWORKERROUTE) Len() int {
	return len(rd.String())
}

func (rd CFWORKERROUTE) String() string {
	return txtutil.Zoneify([]string{rd.When, rd.Then})
}

func MakeCFWORKERROUTE(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 2 {
		return CFWORKERROUTE{}, fmt.Errorf("CF_WORKER_ROUTE expects 2 arguments, got %d: %+v", len(args), args)
	}
	return CFWORKERROUTE{
		When: mustbe.RawString(args[0]),
		Then: mustbe.RawString(args[1]),
	}, nil
}
