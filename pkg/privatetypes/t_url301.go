package privatetypes

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// URL301

func init() {
	Register(TypeURL301, "URL301", func() dnsv2.RR { return new(URL301) }, privatetypesrdata.MakeURL301)
}

const TypeURL301 = uint16(65300)

type URL301 struct {
	Hdr dnsv2.Header

	privatetypesrdata.URL301
	// Location             string
	// PorkbunIncludePath   bool
	// PorkbunWildCard      bool
}

// Typer interface.

func (rr *URL301) Type() uint16 { return TypeURL301 }

// RR interface.

func (rr *URL301) Header() *dnsv2.Header { return &rr.Hdr }
func (rr *URL301) Len() int {
	return rr.Hdr.Len() + rr.Data().Len()
}
func (rr *URL301) Data() dnsv2.RDATA {
	return &privatetypesrdata.URL301{Location: rr.Location, PorkbunIncludePath: rr.PorkbunIncludePath, PorkbunWildCard: rr.PorkbunWildCard}
}
func (rr *URL301) Clone() dnsv2.RR {
	return &URL301{
		Hdr: rr.Hdr,
		URL301: privatetypesrdata.URL301{
			Location: rr.Location,
			PorkbunIncludePath: rr.PorkbunIncludePath,
			PorkbunWildCard: rr.PorkbunWildCard,
		}}
}
func (rr *URL301) String() string {
	return (rr.Header().Name + "\t" +
		strconv.FormatInt(int64(rr.Header().TTL), 10) + "\t" +
		dnsutilv2.ClassToString(rr.Header().Class) + "\tURL301\t" + rr.Data().String())
}

// Parse makes an RDATA for this type using the tokens from dnsv2's parser.
func (rr *URL301) Parse(tokens []string, s string) error {
	args := TokensToArgs(tokens)
	if len(args) != 3 {
		return fmt.Errorf("URL301 requires exactly 3 arguments, got %d: %v", len(args), args)
	}
	rr.Location = mustbe.TargetHost("", args[0])
	rr.PorkbunIncludePath = mustbe.Bool(args[1])
	rr.PorkbunWildCard = mustbe.Bool(args[2])
	return nil
}
