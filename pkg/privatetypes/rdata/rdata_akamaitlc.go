package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type AKAMAITLC struct {
	AnswerType string
	Target     string
}

func (rd AKAMAITLC) Len() int {
	return len(rd.String())
}

func (rd AKAMAITLC) String() string {
	return txtutil.Zoneify([]string{rd.AnswerType, rd.Target})
}

func MakeAKAMAITLC(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 2 {
		return AKAMAITLC{}, fmt.Errorf("AKAMAITLC expects 2 arguments, got %d: %+v", len(args), args)
	}
	return AKAMAITLC{
		AnswerType: mustbe.RawString(args[0]),
		Target:     mustbe.TargetHost(origin, args[1]),
	}, nil
}
