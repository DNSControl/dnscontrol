package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type CLOUDFLAREAPISINGLEREDIRECT struct {
	Name                 string
	Code                 uint16
	When                 string
	Then                 string
}

func (rd CLOUDFLAREAPISINGLEREDIRECT) Len() int {
	return len(rd.String())
}

func (rd CLOUDFLAREAPISINGLEREDIRECT) String() string {
	return txtutil.Zoneify([]string{rd.Name, fmt.Sprintf("%d", rd.Code), rd.When, rd.Then})
}

func MakeCLOUDFLAREAPISINGLEREDIRECT(_ string, args ...any) (dnsv2.RDATA, error) {
	if len(args) != 4 {
		return CLOUDFLAREAPISINGLEREDIRECT{}, fmt.Errorf("CLOUDFLAREAPI_SINGLE_REDIRECT expects 4 arguments, got %d: %+v", len(args), args)
	}
	return CLOUDFLAREAPISINGLEREDIRECT{
		Name: mustbe.RawString(args[0]),
		Code: mustbe.Uint16(args[1]),
		When: mustbe.RawString(args[2]),
		Then: mustbe.RawString(args[3]),
	}, nil
}
