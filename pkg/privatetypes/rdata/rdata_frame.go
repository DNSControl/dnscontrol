package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type FRAME struct {
	Target string
}

func (rd FRAME) Len() int {
	return len(rd.String())
}

func (rd FRAME) String() string {
	return txtutil.Zoneify([]string{rd.Target})
}

func MakeFRAME(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return FRAME{}, fmt.Errorf("FRAME expects 1 arguments, got %d: %+v", len(args), args)
	}
	return FRAME{
		Target: mustbe.RawString(args[0]),
	}, nil
}
