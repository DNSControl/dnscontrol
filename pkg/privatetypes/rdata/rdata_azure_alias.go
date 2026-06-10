package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type AZUREALIAS struct {
	AliasType string
	Target    string
}

func (rd AZUREALIAS) Len() int {
	return len(rd.String())
}

func (rd AZUREALIAS) String() string {
	return txtutil.Zoneify([]string{rd.AliasType, rd.Target})
}

func MakeAZUREALIAS(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 2 {
		return AZUREALIAS{}, fmt.Errorf("AZURE_ALIAS expects 2 arguments, got %d: %+v", len(args), args)
	}
	return AZUREALIAS{
		AliasType: mustbe.RawString(args[0]),
		Target:    mustbe.TargetHost(origin, args[1]),
	}, nil
}
