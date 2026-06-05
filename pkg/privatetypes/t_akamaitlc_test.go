package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

func TestAkamaitlc(t *testing.T) {
	y := &AKAMAITLC{
		Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET},
		AKAMAITLC: privatetypesrdata.AKAMAITLC{
			AnswerType: "cname",
			Target: "example.com.",
		},
	}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("AKAMAITLC string presentations should be identical:\n%s\n%s", rry.String(), y.String())
	}
}
