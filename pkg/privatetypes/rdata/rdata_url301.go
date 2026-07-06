package privatetypesrdata

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

type URL301 struct {
	Location           string
	PorkbunIncludePath bool
	PorkbunWildCard    bool
}

func (rd URL301) Len() int {
	return len(rd.String())
}

func (rd URL301) String() string {
	parts := make([]string, 0, 3)
	parts = append(parts, rd.Location)
	parts = append(parts, fmt.Sprintf("%t", rd.PorkbunIncludePath))
	parts = append(parts, fmt.Sprintf("%t", rd.PorkbunWildCard))
	return strings.Join(parts, " ")
}

func MakeURL301(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)

	// backwards compatibility for providers which aren't porkbun
	if len(args) == 1 {
		return URL301{
			Location:           mustbe.TargetHost(origin, args[0]),
			PorkbunIncludePath: false,
			PorkbunWildCard:    false,
		}, nil
	}

	if len(args) != 3 {
		return nil, fmt.Errorf("URL301 expects 3 arguments, got %d: %+v", len(args), args)
	}
	return URL301{
		Location:           mustbe.TargetHost(origin, args[0]),
		PorkbunIncludePath: mustbe.Bool(args[1]),
		PorkbunWildCard:    mustbe.Bool(args[2]),
	}, nil
}
