package privatetypesrdata

import (
	"fmt"

	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type R53WEIGHTED struct {
	Weight uint8
	SetID  string
}

func (rd R53WEIGHTED) Len() int {
	return len(rd.String())
}

func (rd R53WEIGHTED) String() string {
	parts := make([]string, 0, 2)
	parts = append(parts, fmt.Sprintf("%d", rd.Weight))
	parts = append(parts, txtutil.ZoneifyString(rd.SetID))
	return strings.Join(parts, " ")
}

func MakeR53WEIGHTED(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 2 {
		return nil, fmt.Errorf("R53_WEIGHTED expects 2 arguments, got %d: %+v", len(args), args)
	}
	return &R53WEIGHTED{
		Weight: mustbe.Uint8(args[0]),
		SetID:  mustbe.RawString(args[1]),
	}, nil
}
