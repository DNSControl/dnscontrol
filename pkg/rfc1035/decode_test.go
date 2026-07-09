package rfc1035_test

import (
	"slices"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/rfc1035"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		s    string
		want []string
	}{
		{"a1", `simple`, []string{`simple`}},
		{"a2", `"quoted"`, []string{`quoted`}},
		{"a3", `one two`, []string{`one`, `two`}},
		{"a4", `"buckle" "my" "shoe"`, []string{`buckle`, `my`, `shoe`}},
		{"a5", `"buckle" bare "shoe"`, []string{`buckle`, `bare`, `shoe`}},
		{"a5", `buckle "quoted" shoe`, []string{`buckle`, `quoted`, `shoe`}},
		//
		{"b1", `;`, []string{`;`}},
		{"b2", `a;`, []string{`a;`}},
		{"b3", `;z`, []string{`;z`}},
		{"b4", `a;z`, []string{`a;z`}},
		{"b1q", `";"`, []string{`;`}},
		{"b2q", `"a;"`, []string{`a;`}},
		{"b3q", `";z"`, []string{`;z`}},
		{"b4q", `"a;z"`, []string{`a;z`}},
		//
		{"c1", `\;`, []string{`;`}},
		{"c2", `a\;`, []string{`a;`}},
		{"c3", `\;z`, []string{`;z`}},
		{"c4", `a\;z`, []string{`a;z`}},
		{"c1q", `"\;"`, []string{`;`}},
		{"c2q", `"a\;"`, []string{`a;`}},
		{"c3q", `"\;z"`, []string{`;z`}},
		{"c4q", `"a\;z"`, []string{`a;z`}},
		//
		{"d1", `\j`, []string{`j`}},
		{"d2", `a\j`, []string{`aj`}},
		{"d3", `\jz`, []string{`jz`}},
		{"d4", `a\jz`, []string{`ajz`}},
		{"d1q", `"\j"`, []string{`j`}},
		{"d2q", `"a\j"`, []string{`aj`}},
		{"d3q", `"\jz"`, []string{`jz`}},
		{"d4q", `"a\jz"`, []string{`ajz`}},
		//
		{"e1", `\059`, []string{`;`}},
		{"e2", `a\059`, []string{`a;`}},
		{"e3", `\059z`, []string{`;z`}},
		{"e4", `a\059z`, []string{`a;z`}},
		//
		{"f1", `\109`, []string{`m`}},
		{"f2", `a\109`, []string{`am`}},
		{"f3", `\109z`, []string{`mz`}},
		{"f4", `a\109z`, []string{`amz`}},
		//
		{"g1", `\003`, []string{"\x03"}},
		{"g2", `a\003`, []string{"a\x03"}},
		{"g3", `\003z`, []string{"\x03z"}},
		{"g4", `a\003z`, []string{"a\x03z"}},
		//
		{"h1", `\127`, []string{"\x7f"}},
		{"h2", `a\127`, []string{"a\x7f"}},
		{"h3", `\127z`, []string{"\x7fz"}},
		{"h4", `a\127z`, []string{"a\x7fz"}},
		// Special cases
		{"amazon", `"first""second"`, []string{`first`, `second`}},
		{"unclosedquote", `"unclosedquote`, []string{`unclosedquote`}},
		{"unstartedquote", `unclosedquote"`, []string{`unclosedquote"`}},
		// Would be errors, but we just press on.
		{"i1", `\`, []string{`\`}},
		{"i2", `\1`, []string{`1`}},
		{"i3", `\10`, []string{`10`}},
		{"i4", `a\`, []string{`a\`}},
		{"i5", `a\1`, []string{`a1`}},
		{"i6", `a\10`, []string{`a10`}},
		{"i7", `\1f`, []string{`1f`}},
		{"i8", `\1fo`, []string{`1fo`}},
		{"i9", `\1foo`, []string{`1foo`}},
		{"i10", `\1fooo`, []string{`1fooo`}},
		{"i11", `\10f`, []string{`10f`}},
		{"i12", `\10fo`, []string{`10fo`}},
		{"i13", `\10foo`, []string{`10foo`}},
		{"i14", `\10fooo`, []string{`10fooo`}},
		// fun with backslashes
		{`back1`, `\`, []string{`\`}},
		{`back2`, `\\`, []string{`\`}},
		{`back3`, `\\\`, []string{`\\`}},
		{`back4`, `\\\\`, []string{`\\`}},
		{`back5`, `\\\\\`, []string{`\\\`}},
		{`aback1`, `a\`, []string{`a\`}},
		{`aback2`, `a\\`, []string{`a\`}},
		{`aback3`, `a\\\`, []string{`a\\`}},
		{`aback4`, `a\\\\`, []string{`a\\`}},
		{`aback5`, `a\\\\\`, []string{`a\\\`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rfc1035.Decode(tt.s)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Decode(%s) = %v, want %v", tt.s, got, tt.want)
			}

			// Round-trip:
			encoded := rfc1035.Encode(got)
			decoded := rfc1035.Decode(encoded)
			encoded2 := rfc1035.Encode(decoded)
			if !slices.Equal(got, decoded) || encoded != encoded2 {
				t.Errorf("roundtrip failed: input=%s orig=%s encoded=%s decoded=%s", tt.s, got, encoded, decoded)
			}

		})
	}
}

func TestIsRemaining(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		s    string
		i    int
		r    int
		want bool
	}{
		{"a1", "012345", 0, 3, true},
		{"a2", "012345", 1, 3, true},
		{"a3", "012345", 2, 3, true},
		{"a4", "012345", 3, 3, false},
		{"a5", "012345", 4, 3, false},
		{"a6", "012345", 5, 3, false},
		{"a7", "012345", 6, 3, false},
		{"a8", "012345", 7, 3, false},
		//
		{"j1", `\j`, 0, 1, true},
		{"j2", `\j`, 0, 3, false},
		{"x1", `xj`, 0, 1, true},
		{"x2", `xj`, 0, 3, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rfc1035.IsRemaining(tt.s, tt.i, tt.r)
			if got != tt.want {
				t.Errorf("IsRemaining(%q, %d, %d) = %v, want %v", tt.s, tt.i, tt.r, got, tt.want)
			}
		})
	}
}
