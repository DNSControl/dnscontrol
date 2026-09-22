package scaleway

import (
	"strings"
	"testing"
)

// bindUnescape mimics Scaleway's write-side handling: it takes the BIND-style
// quoted value we send and returns the raw bytes Scaleway stores (strip the
// surrounding quotes, then process `\` escapes).
func bindUnescape(s string) string {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return s
	}
	inner := s[1 : len(s)-1]
	var b strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) {
			i++
		}
		b.WriteByte(inner[i])
	}
	return b.String()
}

// fakeScalewayRoundTrip simulates the full server round-trip for a TXT value:
// we send quoteTXT(value); Scaleway unescapes and stores it; on read it wraps
// the raw bytes in quotes again, escaping interior double quotes as `\"` but
// leaving backslashes verbatim.
func fakeScalewayRoundTrip(value string) string {
	stored := bindUnescape(quoteTXT(value))
	returned := strings.ReplaceAll(stored, `"`, `\"`)
	return `"` + returned + `"`
}

// TestTXTRoundTrip checks that a TXT value survives the trip through Scaleway.
// Values that AuditRecords rejects are left out: they never reach this code.
func TestTXTRoundTrip(t *testing.T) {
	cases := []string{
		"hello",
		`a "quoted" string`,
		`in"side`,
		`in"ter"ior`,
		`1back\slash`,
		`2back\\slash`,
		`3back\\\slash`,
		`4back\\\\slash`,
		"v=spf1 include:_spf.example.com ~all",
		"with spaces",
	}
	for _, in := range cases {
		returned := fakeScalewayRoundTrip(in)
		out, err := unquoteTXT(returned)
		if err != nil {
			t.Fatalf("unquoteTXT(%q) err: %v", returned, err)
		}
		if out != in {
			t.Errorf("round-trip mismatch: in=%q sent=%q returned=%q out=%q",
				in, quoteTXT(in), returned, out)
		}
	}
}

func TestUnquoteTXTUnquoted(t *testing.T) {
	// Inputs without surrounding quotes should pass through unchanged.
	in := "no-quotes here"
	out, err := unquoteTXT(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != in {
		t.Errorf("expected passthrough, got %q", out)
	}
}
