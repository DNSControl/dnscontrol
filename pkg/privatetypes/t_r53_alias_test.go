package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

func TestR53Alias_NormalUser(t *testing.T) {
	y := &R53ALIAS{
		Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET},
		R53ALIAS: privatetypesrdata.R53ALIAS{
			AliasType:        "1",
			Target:           "alice.",
			EvalTargetHealth: "true",
			ZoneID:           "1234",
		},
	}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("R53_ALIAS string presentations should be identical:\n%s\n%s", rry.String(), y.String())
	}
}
