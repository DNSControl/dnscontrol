package privatetypesrdata

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/rfc1035"
)

type R53ALIAS struct {
	AliasType        string
	Target           string
	EvalTargetHealth string
	ZoneID           string
}

func (rd R53ALIAS) Len() int {
	return len(rd.String())
}

func (rd R53ALIAS) String() string {
	parts := make([]string, 0, 4)
	parts = append(parts, rfc1035.EncodeString(rd.AliasType))
	parts = append(parts, rd.Target)
	parts = append(parts, rfc1035.EncodeString(rd.EvalTargetHealth))
	parts = append(parts, rfc1035.EncodeString(rd.ZoneID))
	return strings.Join(parts, " ")
}

func MakeR53ALIAS(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) < 2 || len(args) > 4 {
		return nil, fmt.Errorf("R53_ALIAS expects 4 arguments, got %d: %+v", len(args), args)
	}
	for len(args) < 4 {
		args = append(args, "")
	}
	return R53ALIAS{
		AliasType:        mustbe.RawString(args[0]),
		Target:           mustbe.TargetHost(origin, args[1]),
		EvalTargetHealth: mustbe.RawString(args[2]),
		ZoneID:           mustbe.RawString(args[3]),
	}, nil
}
