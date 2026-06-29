package privatetypesrdata

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

type BUNNYDNSPZ struct {
	PullZoneID int64
}

func (rd BUNNYDNSPZ) Len() int {
	return len(rd.String())
}

func (rd BUNNYDNSPZ) String() string {
	parts := make([]string, 0, 1)
	parts = append(parts, fmt.Sprintf("%d", rd.PullZoneID))
	return strings.Join(parts, " ")
}

func MakeBUNNYDNSPZ(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("BUNNY_DNS_PZ expects 1 arguments, got %d: %+v", len(args), args)
	}
	return BUNNYDNSPZ{
		PullZoneID: mustbe.Int64(args[0]),
	}, nil
}
