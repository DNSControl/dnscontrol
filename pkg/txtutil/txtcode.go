package txtutil

import (
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
)

// ParseQuoted parses a string of RFC1035-style quoted items. The resulting
// items are then joined into one string. This is useful for parsing TXT
// records.
// Examples:
// `foo` => foo
// `"foo"` => foo
// `"f\"oo"` => f"oo
// `"f\\oo"` => f\oo
// `"foo" "bar"` => foobar
// `"foo" bar` => foobar.
func ParseQuoted(s string) (string, error) {
	return txtDecode(s)
}

// EncodeQuoted encodes a string into a series of quoted 255-octet chunks. That
// is, when decoded each chunk would be 255-octets with the remainder in the
// last chunk.
//
// The output looks like:
//
//	`""`                                      empty
//	`"255\"octets"`                           quotes are escaped
//	`"255\\octets"`                           backslashes are escaped
//	`"255octets" "255octets" "remainder"`     long strings are chunked
func EncodeQuoted(t string) string {
	return txtEncode(ToChunks(t))
}

// EncodeSingle encodes a string as a single quoted value without splitting
// into 255-octet chunks. This is intended for user-facing display (e.g., diff
// preview) where the chunked representation is confusing.
func EncodeSingle(t string) string {
	return txtEncode([]string{t})
}

// txtDecode decodes TXT strings quoted/escaped as Tom interprets RFC10225.
func txtDecode(s string) (string, error) {

	rd, err := dnsv2.NewData(dnsv2.TypeTXT, s)
	if err != nil {
		return "", err
	}
	rdtxt := rd.(dnsrdatav2.TXT)
	return strings.Join(rdtxt.Txt, ""), nil
}

// txtEncode encodes TXT strings in RFC1035 format as interpreted by Tom.
func txtEncode(ts []string) string {
	// printer.Printf("DEBUG: txtEncode txt outboundv=%v\n", ts)
	if (len(ts) == 0) || (strings.Join(ts, "") == "") {
		return `""`
	}

	rd := dnsrdatav2.TXT{Txt: ts}
	t := rd.String()

	// printer.Printf("DEBUG: txtEncode txt  encodedv=%v\n", t)
	return t
}
