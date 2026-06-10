package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

func TestCloudflareapiSingleRedirect(t *testing.T) {
	y := &CLOUDFLAREAPISINGLEREDIRECT{
		Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET},
		CLOUDFLAREAPISINGLEREDIRECT: privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT{
			SRName: "",
			Code:   301,
			SRWhen: "",
			SRThen: "",
		},
	}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("CLOUDFLAREAPI_SINGLE_REDIRECT string presentations should be identical:\n%s\n%s", rry.String(), y.String())
	}
}
