package privatetypesrdata

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type URL struct {
	Location           string
	PorkbunIncludePath bool
	PorkbunWildCard    bool
}

func (rd URL) Len() int {
	return len(rd.String())
}

func (rd URL) String() string {
	parts := make([]string, 0, 3)
	parts = append(parts, txtutil.ZoneifyString(rd.Location))
	parts = append(parts, fmt.Sprintf("%t", rd.PorkbunIncludePath))
	parts = append(parts, fmt.Sprintf("%t", rd.PorkbunWildCard))
	return strings.Join(parts, " ")
}

func MakeURL(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 3 {
		return nil, fmt.Errorf("URL expects 3 arguments, got %d: %+v", len(args), args)
	}
	return URL{
		Location:           mustbe.RawString(args[0]),
		PorkbunIncludePath: mustbe.Bool(args[1]),
		PorkbunWildCard:    mustbe.Bool(args[2]),
	}, nil
}
