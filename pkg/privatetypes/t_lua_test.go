package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

func TestLua(t *testing.T) {
	y := &LUA{
		Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET},
		LUA: privatetypesrdata.LUA{
			LuaType:    "A",
			LuaPayload: "return_127_0_0_1",
		},
	}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("LUA string presentations should be identical:\n%s\n%s", rry.String(), y.String())
	}
}
