package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
)

type MIKROTIKNXDOMAIN struct {
}

func (rd MIKROTIKNXDOMAIN) Len() int {
	return 0
}

func (rd MIKROTIKNXDOMAIN) String() string {
	return ""
}

func MakeMIKROTIKNXDOMAIN(origin string, args []any, _ map[string]string) (dnsv2.RDATA, error) {
	if len(args) != 0 {
		return MIKROTIKNXDOMAIN{}, fmt.Errorf("MIKROTIK_NXDOMAIN expects 0 arguments, got %d: %+v", len(args), args)
	}
	return MIKROTIKNXDOMAIN{}, nil
}
