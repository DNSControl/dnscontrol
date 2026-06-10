package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type CLOUDFLAREAPISINGLEREDIRECT struct {
	SRName string
	Code   uint16
	SRWhen string
	SRThen string
}

func (rd CLOUDFLAREAPISINGLEREDIRECT) Len() int {
	return len(rd.String())
}

func (rd CLOUDFLAREAPISINGLEREDIRECT) String() string {
	return txtutil.Zoneify([]string{rd.SRName, fmt.Sprintf("%d", rd.Code), rd.SRWhen, rd.SRThen})
}

func MakeCLOUDFLAREAPISINGLEREDIRECT(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return CLOUDFLAREAPISINGLEREDIRECT{}, fmt.Errorf("CLOUDFLAREAPI_SINGLE_REDIRECT expects 4 arguments, got %d: %+v", len(args), args)
	}
	return CLOUDFLAREAPISINGLEREDIRECT{
		SRName: mustbe.RawString(args[0]),
		Code:   mustbe.Uint16(args[1]),
		SRWhen: mustbe.RawString(args[2]),
		SRThen: mustbe.RawString(args[3]),
	}, nil
}
