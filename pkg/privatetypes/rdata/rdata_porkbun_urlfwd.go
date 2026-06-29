package privatetypesrdata

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type PORKBUNURLFWD struct {
	Target      string
	TypeName    string
	IncludePath string
	Wildcard    string
}

func (rd PORKBUNURLFWD) Len() int {
	return len(rd.String())
}

func (rd PORKBUNURLFWD) String() string {
	parts := make([]string, 0, 4)
	parts = append(parts, txtutil.ZoneifyString(rd.Target))
	parts = append(parts, txtutil.ZoneifyString(rd.TypeName))
	parts = append(parts, txtutil.ZoneifyString(rd.IncludePath))
	parts = append(parts, txtutil.ZoneifyString(rd.Wildcard))
	return strings.Join(parts, " ")
}

func MakePORKBUNURLFWD(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return nil, fmt.Errorf("PORKBUN_URLFWD expects 4 arguments, got %d: %+v", len(args), args)
	}
	return PORKBUNURLFWD{
		Target:      mustbe.RawString(args[0]),
		TypeName:    mustbe.RawString(args[1]),
		IncludePath: mustbe.RawString(args[2]),
		Wildcard:    mustbe.RawString(args[3]),
	}, nil
}
