package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

func TestUrl(t *testing.T) {
	y := &URL{
		Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET},
		URL: privatetypesrdata.URL{
			Location: "https://example.com/",
			PorkbunIncludePath: true,
			PorkbunWildCard: false,
		},
	}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("URL string presentations should be identical:\n%s\n%s", rry.String(), y.String())
	}
}
