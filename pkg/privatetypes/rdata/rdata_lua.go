package privatetypesrdata

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type LUA struct {
	LuaType    string
	LuaPayload string
}

func (rd LUA) Len() int {
	return len(rd.String())
}

func (rd LUA) String() string {
	return txtutil.Zoneify([]string{rd.LuaType, rd.LuaPayload})
}

func MakeLUA(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 2 {
		return LUA{}, fmt.Errorf("LUA expects 2 arguments, got %d: %+v", len(args), args)
	}
	return LUA{
		LuaType:    mustbe.RawString(args[0]),
		LuaPayload: mustbe.RawString(args[1]),
	}, nil
}
