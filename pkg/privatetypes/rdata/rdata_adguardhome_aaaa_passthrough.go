package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

type ADGUARDHOMEAAAAPASSTHROUGH struct {
}

func (rd ADGUARDHOMEAAAAPASSTHROUGH) Len() int {
	return 0
}

func (rd ADGUARDHOMEAAAAPASSTHROUGH) String() string {
	return ""
}

func MakeADGUARDHOMEAAAAPASSTHROUGH(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 0 {
		return ADGUARDHOMEAAAAPASSTHROUGH{}, fmt.Errorf("ADGUARDHOME_AAAA_PASSTHROUGH expects 0 arguments, got %d: %+v", len(args), args)
	}
	return ADGUARDHOMEAAAAPASSTHROUGH{}, nil
}
