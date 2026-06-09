package privatetypes

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
)

// TypeToMakeRDATA returns a function that accepts arguments of any type and returns a dnsv2.RDATA struct.
// Examples:
//
//	demoRC1, err := TypeToMakeRDATA[dnsv2.TypeA](mustbe.IPv4("1.2.3.4"))
//	demoRC2, err := TypeToMakeRDATA[dnsv2.TypeCNAME](mustbe.Host("www", "example.com"))
//	demoRC3, err := TypeToMakeRDATA[privatetypes.TypeCFWORKERROUTE](mustbe.RawString("example.com/*"), mustbe.RawString("example.com/worker"))

type MakerRn func(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error)

var TypeToMakeRDATA = make(map[uint16]MakerRn)

// Register registers a new private RR type. It panics if the code point or name is already in use.
func Register(codepoint uint16, typeName string, newFn func() dnsv2.RR, makeFn MakerRn) {

	/*
		# Private Resource Records

		Any struct can be used as a private resource record. To make it work you need to implement the following interfaces.

		  - [Typer], to give your RR a code point, and see documentation of that interface.
		  - [RR], all RRs implement this, if you want to have a private EDNS0 option, implement [EDNS0] interface, this
		    adds a Pseudo() bool method.
		  - [Parser], so it can be parsed to and from strings.
		  - [Packer], if you need to use your new RR on the wire.
		  - [Comparer], if your RR will be signed with DNSSEC.

		See rr_test.go for a complete example for both an external [RR] and [EDNS0].
	*/

	// typenum -> func() RR  i.e. a function that creates a new RR struct for the given code point.
	if dnsv2.TypeToRR[codepoint] != nil {
		panic(fmt.Sprintf("TypeToRR[%d] already in use (check for duplicate codepoint assignments)", codepoint))
	}
	dnsv2.TypeToRR[codepoint] = newFn

	// typenum -> typename
	if dnsv2.TypeToString[codepoint] != "" {
		panic(fmt.Sprintf("TypeToString[%d] already in use by %s", codepoint, dnsv2.TypeToString[codepoint]))
	}
	dnsv2.TypeToString[codepoint] = typeName

	// typename -> typenum
	if s, exists := dnsv2.StringToType[typeName]; exists {
		panic(fmt.Sprintf("StringToType[%s] already in use by %d", typeName, s))
	}
	dnsv2.StringToType[typeName] = codepoint

	RegisterMaker(codepoint, makeFn)
}

// RegisterMaker registers just the Make*() function for an rtype. This is needed for non-private types.
func RegisterMaker(codepoint uint16, makeFn MakerRn) {

	typeName := dnsv2.TypeToString[codepoint]

	// typenum -> func(args ...any) (RDATA, error) i.e. a function that creates an RDATA struct for the given code point, with fields filled from the given args.
	if s, exists := TypeToMakeRDATA[codepoint]; exists {
		panic(fmt.Sprintf("TypeToMakeRDATA[%d] a.k.a. %s already in use by %T", codepoint, typeName, s))
	}
	TypeToMakeRDATA[codepoint] = makeFn
}

func IsModernType(typeName string) bool {
	_, ok := TypeToMakeRDATA[dnsv2.StringToType[typeName]]
	return ok
}
