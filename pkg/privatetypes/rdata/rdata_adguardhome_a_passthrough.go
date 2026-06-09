package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
)

type ADGUARDHOMEAPASSTHROUGH struct {
}

func (rd ADGUARDHOMEAPASSTHROUGH) Len() int {
	return 0
}

func (rd ADGUARDHOMEAPASSTHROUGH) String() string {
	return ""
}

func MakeADGUARDHOMEAPASSTHROUGH(origin string, args []any, _ map[string]string) (dnsv2.RDATA, error) {
	if len(args) != 0 {
		return ADGUARDHOMEAPASSTHROUGH{}, fmt.Errorf("ADGUARDHOME_A_PASSTHROUGH expects 0 arguments, got %d: %+v", len(args), args)
	}
	return ADGUARDHOMEAPASSTHROUGH{}, nil
}
