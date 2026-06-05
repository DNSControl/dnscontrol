package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
)

func TestMikrotikNxdomain(t *testing.T) {
	y := &MIKROTIKNXDOMAIN{Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET}}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("MIKROTIK_NXDOMAIN string presentations should be identical:\n%q\n%q", rry.String(), y.String())
	}
}
