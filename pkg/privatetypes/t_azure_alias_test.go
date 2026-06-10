package privatetypes

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

func TestAzureAlias(t *testing.T) {
	y := &AZUREALIAS{
		Hdr: dnsv2.Header{Name: "example.org.", Class: dnsv2.ClassINET},
		AZUREALIAS: privatetypesrdata.AZUREALIAS{
			AliasType: "A",
			Target:    "example.com.",
		},
	}
	rry, err := dnsv2.New(y.String())
	if err != nil {
		t.Fatal(err)
	}
	if rry.String() != y.String() {
		t.Fatalf("AZURE_ALIAS string presentations should be identical:\n%s\n%s", rry.String(), y.String())
	}
}
