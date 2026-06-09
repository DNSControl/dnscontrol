package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type CLOUDFLAREAPISINGLEREDIRECT struct {
	Code                 uint16
	SRName               string
	SRWhen               string
	SRThen               string
}

func (rd CLOUDFLAREAPISINGLEREDIRECT) Len() int {
	return len(rd.String())
}

func (rd CLOUDFLAREAPISINGLEREDIRECT) String() string {
	return txtutil.Zoneify([]string{fmt.Sprintf("%d", rd.Code), rd.SRName, rd.SRWhen, rd.SRThen})
}

func MakeCLOUDFLAREAPISINGLEREDIRECT(origin string, args []any, _ map[string]string) (dnsv2.RDATA, error) {
	if len(args) != 4 {
		return CLOUDFLAREAPISINGLEREDIRECT{}, fmt.Errorf("CLOUDFLAREAPI_SINGLE_REDIRECT expects 4 arguments, got %d: %+v", len(args), args)
	}
	return CLOUDFLAREAPISINGLEREDIRECT{
		Code: mustbe.Uint16(args[0]),
		SRName: mustbe.RawString(args[1]),
		SRWhen: mustbe.RawString(args[2]),
		SRThen: mustbe.RawString(args[3]),
	}, nil
}
