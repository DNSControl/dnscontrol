package scaleway

import "testing"

func TestQuoteUnquoteTXT(t *testing.T) {
	cases := []string{
		"",
		"hello",
		`a "quoted" string`,
		`back\slash`,
		"v=spf1 include:_spf.example.com ~all",
	}
	for _, in := range cases {
		q := quoteTXT(in)
		out, err := unquoteTXT(q)
		if err != nil {
			t.Fatalf("unquoteTXT(%q) err: %v", q, err)
		}
		if out != in {
			t.Errorf("round-trip mismatch: in=%q quoted=%q out=%q", in, q, out)
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
