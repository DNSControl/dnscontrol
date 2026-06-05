package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type R53ALIAS struct {
	AliasType            string
	Target               string
	EvalTargetHealth     string
	ZoneID               string
}

func (rd R53ALIAS) Len() int {
	return len(rd.AliasType) +
		1 + len(rd.Target) +
		1 + len(rd.EvalTargetHealth) +
		1 + len(rd.ZoneID)
}

func (rd R53ALIAS) String() string {
	return txtutil.Zoneify([]string{rd.AliasType, rd.Target, rd.EvalTargetHealth, rd.ZoneID})
}

func MakeR53ALIAS(origin string, args ...any) (dnsv2.RDATA, error) {
	if len(args) != 4 {
		return R53ALIAS{}, fmt.Errorf("R53_ALIAS expects 4 arguments, got %d: %+v", len(args), args)
	}
	return R53ALIAS{mustbe.RawString(args[0]), mustbe.TargetHost("", args[1]), mustbe.RawString(args[2]), mustbe.RawString(args[3])}, nil
}
