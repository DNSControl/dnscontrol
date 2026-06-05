package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
)

type PORKBUNURLFWD struct {
}

func (rd PORKBUNURLFWD) Len() int {
	return 0
}

func (rd PORKBUNURLFWD) String() string {
	return ""
}

func MakePORKBUNURLFWD(origin string, args ...any) (dnsv2.RDATA, error) {
	if len(args) != 0 {
		return PORKBUNURLFWD{}, fmt.Errorf("PORKBUN_URLFWD expects 0 arguments, got %d: %+v", len(args), args)
	}
	return PORKBUNURLFWD{}, nil
}
