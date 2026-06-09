package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

type ADGUARDHOMEAPASSTHROUGH struct {
}

func (rd ADGUARDHOMEAPASSTHROUGH) Len() int {
	return 0
}

func (rd ADGUARDHOMEAPASSTHROUGH) String() string {
	return ""
}

func MakeADGUARDHOMEAPASSTHROUGH(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 0 {
		return ADGUARDHOMEAPASSTHROUGH{}, fmt.Errorf("ADGUARDHOME_A_PASSTHROUGH expects 0 arguments, got %d: %+v", len(args), args)
	}
	return ADGUARDHOMEAPASSTHROUGH{}, nil
}
