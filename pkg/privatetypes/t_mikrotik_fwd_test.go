package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

func TestMikrotikFwd(t *testing.T) {
	y := &MIKROTIKFWD{
		Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET},
		MIKROTIKFWD: privatetypesrdata.MIKROTIKFWD{
			ForwardTo: "example.com.",
		},
	}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("MIKROTIK_FWD string presentations should be identical:\n%s\n%s", rry.String(), y.String())
	}
}
