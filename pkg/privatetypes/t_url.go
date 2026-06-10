package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// URL

func init() {
	Register(TypeURL, "URL", func() dnsv2.RR { return new(URL) }, privatetypesrdata.MakeURL)
}

const TypeURL = uint16(65299)

type URL struct {
	Hdr dnsv2.Header

	privatetypesrdata.URL
	// Location             string
	// PorkbunIncludePath   bool
	// PorkbunWildCard      bool
}

// Typer interface.

func (rr *URL) Type() uint16 { return TypeURL }

// RR interface.

func (rr *URL) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *URL) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *URL) Data() dnsv2.RDATA {
	return &privatetypesrdata.URL{Location: rr.Location, PorkbunIncludePath: rr.PorkbunIncludePath, PorkbunWildCard: rr.PorkbunWildCard}
}
func (rr *URL) Clone() dnsv2.RR {
	return &URL{
		Hdr: rr.Hdr,
		URL: privatetypesrdata.URL{
			Location:           rr.Location,
			PorkbunIncludePath: rr.PorkbunIncludePath,
			PorkbunWildCard:    rr.PorkbunWildCard,
		}}
}
func (rr *URL) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tURL\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *URL) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 3 {
		return fmt.Errorf("URL requires exactly 3 arguments, got %d: %v", len(args), args)
	}
	rr.Location = mustbe.RawString(args[0])
	rr.PorkbunIncludePath = mustbe.Bool(args[1])
	rr.PorkbunWildCard = mustbe.Bool(args[2])
	return nil
}
