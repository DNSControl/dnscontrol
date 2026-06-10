package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// LUA

func init() {
	Register(TypeLUA, "LUA", func() dnsv2.RR { return new(LUA) }, privatetypesrdata.MakeLUA)
}

const TypeLUA = uint16(65292)

type LUA struct {
	Hdr dnsv2.Header

	privatetypesrdata.LUA
	// LuaType              string
	// LuaPayload           string
}

// Typer interface.

func (rr *LUA) Type() uint16 { return TypeLUA }

// RR interface.

func (rr *LUA) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *LUA) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *LUA) Data() dnsv2.RDATA {
	return &privatetypesrdata.LUA{LuaType: rr.LuaType, LuaPayload: rr.LuaPayload}
}
func (rr *LUA) Clone() dnsv2.RR {
	return &LUA{
		Hdr: rr.Hdr,
		LUA: privatetypesrdata.LUA{
			LuaType: rr.LuaType,
			LuaPayload: rr.LuaPayload,
		}}
}
func (rr *LUA) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tLUA\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *LUA) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 2 {
		return fmt.Errorf("LUA requires exactly 2 arguments, got %d: %v", len(args), args)
	}
	rr.LuaType = mustbe.RawString(args[0])
	rr.LuaPayload = mustbe.RawString(args[1])
	return nil
}
