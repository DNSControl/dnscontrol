package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
)

func TestBunnyDnsRdr(t *testing.T) {
	y := &BUNNYDNSRDR{Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET}}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("BUNNY_DNS_RDR string presentations should be identical:\n%q\n%q", rry.String(), y.String())
	}
}
