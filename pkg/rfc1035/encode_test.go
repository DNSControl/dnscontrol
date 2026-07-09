package rfc1035_test

import (
	"slices"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/rfc1035"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		txts []string
		want string
	}{
		{"a1", []string{`simple`}, `simple`},
		{"a2", []string{`with space`}, `"with space"`},
		{"a3", []string{`one`, `two`}, `"one" "two"`},
		{"a4", []string{`t"wo`}, `"t\"wo"`},
		{"a5", []string{`!@#$%^&*();:'"<>,./?~`}, `"\!\@\#\$\%\^\&*\(\)\;\:\'\"\<\>\,.\/\?\~"`},
		{"a6", []string{"`"}, "\"\\`\""},
		//
		{"b1", []string{"\x03"}, `"\003"`},
		{"b2", []string{"a\x03"}, `"a\003"`},
		{"b3", []string{"\x03z"}, `"\003z"`},
		{"b4", []string{"a\x03z"}, `"a\003z"`},
		///
		{"b1", []string{"\x7f"}, `"\127"`},
		{"b2", []string{"a\x7f"}, `"a\127"`},
		{"b3", []string{"\x7fz"}, `"\127z"`},
		{"b4", []string{"a\x7fz"}, `"a\127z"`},
		///
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rfc1035.Encode(tt.txts)
			if got != tt.want {
				t.Errorf("Encode(%s) = %s, want %s", tt.txts, got, tt.want)
			}

			// Round-trip:
			decoded := rfc1035.Decode(got)
			encoded := rfc1035.Encode(decoded)
			decoded2 := rfc1035.Decode(encoded)
			// encoded3 := rfc1035.Encode(decoded2)
			if !slices.Equal(decoded, decoded2) || got != encoded {
				t.Errorf("roundtrip failed: input=%s encode=%s decode=%s encode=%s decode=%s", tt.txts, got, decoded, encoded, decoded2)
			}
		})
	}
}
