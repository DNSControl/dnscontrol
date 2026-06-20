package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

func TestR53Weighted(t *testing.T) {
	y := &R53WEIGHTED{
		Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET},
		R53WEIGHTED: privatetypesrdata.R53WEIGHTED{
			Weight: 123,
			SetID:  "abc",
		},
	}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("R53_WEIGHTED string presentations should be identical:\n%s\n%s", rry.String(), y.String())
	}
}
