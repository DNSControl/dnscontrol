package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
)

func TestPorkbunUrlfwd(t *testing.T) {
	y := &PORKBUNURLFWD{Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET}}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("PORKBUN_URLFWD string presentations should be identical:\n%q\n%q", rry.String(), y.String())
	}
}
