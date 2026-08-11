package models

import (
	"fmt"
	"strings"

	dnsv1 "github.com/miekg/dns"
)

/*
TODO(tlim): Move this file to pkgs/txtutil. It doesn't need to be part
*/

// isQuoted returns true if the string starts and ends with a double quote.
func isQuoted(s string) bool {
	if s == "" {
		return false
	}
	if len(s) < 2 {
		return false
	}
	if s[0] == '"' && s[len(s)-1] == s[0] {
		return true
	}
	return false
}

// StripQuotes returns the string with the starting and ending quotes removed.
// If it is not quoted, the original string is returned.
func StripQuotes(s string) string {
	if isQuoted(s) {
		return s[1 : len(s)-1]
	}
	return s
}

// ParseQuotedTxt returns the individual strings of a combined quoted string.
//
//	`foo`  -> []string{"foo"}
//	`"foo"` -> []string{"foo"}
//	`"foo" "bar"` -> []string{"foo", "bar"}
//	`"f"oo" "bar"` -> []string{`f"oo`, "bar"}
//
// NOTE: It is assumed there is exactly one space between the quotes.
// NOTE: This doesn't handle escaped quotes.
// NOTE: You probably want to use ParseQuotedFields() for RFC 1035-compliant quoting.
func ParseQuotedTxt(s string) []string {
	if !isQuoted(s) {
		return []string{s}
	}
	return strings.Split(StripQuotes(s), `" "`)
}

// ParseQuotedFields is like strings.Fields except individual fields
// might be quoted using `"`.
func ParseQuotedFields(s string) ([]string, error) {
	// Parse according to RFC1035 zonefile specifications.
	// "foo"  -> one string: `foo``
	// "foo" "bar"  -> two strings: `foo` and `bar`

	// The dns package doesn't expose the quote parser. Therefore we create a TXT record and extract the strings.
	rr, err := dnsv1.NewRR("example.com. IN TXT " + s)
	if err != nil {
		return nil, fmt.Errorf("could not parse %q TXT: %w", s, err)
	}

	return rr.(*dnsv1.TXT).Txt, nil
}
