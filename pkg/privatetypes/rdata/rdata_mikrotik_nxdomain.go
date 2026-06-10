package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

type MIKROTIKNXDOMAIN struct {
}

func (rd MIKROTIKNXDOMAIN) Len() int {
	return 0
}

func (rd MIKROTIKNXDOMAIN) String() string {
	return ""
}

func MakeMIKROTIKNXDOMAIN(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 0 {
		return MIKROTIKNXDOMAIN{}, fmt.Errorf("MIKROTIK_NXDOMAIN expects 0 arguments, got %d: %+v", len(args), args)
	}
	return MIKROTIKNXDOMAIN{}, nil
}
