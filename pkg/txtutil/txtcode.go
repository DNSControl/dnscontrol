//go:generate go tool stringer -type=State

package txtutil

import (
	"bytes"
	"fmt"
	"strings"
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

// State denotes the parser state.
type State int

const (
	// StateStart indicates parser is looking for a non-space.
	StateStart State = iota

	// StateUnquoted indicates parser is in a run of unquoted text.
	StateUnquoted

	// StateQuoted indicates parser is in quoted text.
	StateQuoted

	// StateBackslash indicates the last char was backlash in a quoted string.
	StateBackslash

	// StateWantSpace indicates parser expects a space (the previous token was a closing quote).
	StateWantSpace
)

func isRemaining(s string, i, r int) bool {
	return (len(s) - 1 - i) > r
}

// txtDecode decodes TXT strings quoted/escaped as Tom interprets RFC10225.
func txtDecode(s string) (string, error) {
	// Parse according to RFC1035 zonefile specifications.
	// "foo"  -> one string: `foo``
	// "foo" "bar"  -> two strings: `foo` and `bar`
	// quotes and backslashes are escaped using \

	/*

		BNF:
			txttarget := `""`` | item | item ` item*
			item := quoteditem | unquoteditem
			quoteditem := quote innertxt quote
			:= `"`
			innertxt := (escaped | printable )*
			escaped := `\\` | `\"`
			printable := (printable ASCII chars)
			unquoteditem := (printable ASCII chars but not `"` nor ' ')

	*/

	// printer.Printf("DEBUG: txtDecode txt inboundv=%v\n", s)

	b := &bytes.Buffer{}
	state := StateStart
	for i, c := range s {
		// printer.Printf("DEBUG: state=%v rune=%v\n", state, string(c))

		switch state {
		case StateStart:
			switch c {
			case ' ':
				// skip whitespace
			case '"':
				state = StateQuoted
			default:
				state = StateUnquoted
				b.WriteRune(c)
			}

		case StateUnquoted:

			if c == ' ' {
				state = StateStart
			} else {
				b.WriteRune(c)
			}

		case StateQuoted:
			switch c {
			case '\\':
				if isRemaining(s, i, 1) {
					state = StateBackslash
				} else {
					return "", fmt.Errorf("txtDecode quoted string ends with backslash q(%q)", s)
				}
			case '"':
				state = StateWantSpace
			default:
				b.WriteRune(c)
			}

		case StateBackslash:
			b.WriteRune(c)
			state = StateQuoted

		case StateWantSpace:
			switch c {
			case ' ':
				state = StateStart
			case '"':
				// Tolerate adjacent quoted character-strings without a
				// separating space (e.g. `"foo""bar"`). Route 53 has been
				// observed to return long TXT records in this form.
				// Whether or not this is valid is questionable but we'll accept it because... Amazon.
				state = StateQuoted
			default:
				return "", fmt.Errorf("txtDecode expected whitespace after close quote q(%q)", s)
			}
		}
	}

	r := b.String()
	// printer.Printf("DEBUG: txtDecode txt decodedv=%v\n", r)
	return r, nil
}

// txtEncode encodes TXT strings in RFC1035 format as interpreted by Tom.
func txtEncode(ts []string) string {
	// printer.Printf("DEBUG: txtEncode txt outboundv=%v\n", ts)
	if (len(ts) == 0) || (strings.Join(ts, "") == "") {
		return `""`
	}

	var r []string

	for i := range ts {
		tx := ts[i]
		tx = strings.ReplaceAll(tx, `\`, `\\`)
		tx = strings.ReplaceAll(tx, `"`, `\"`)
		tx = `"` + tx + `"`
		r = append(r, tx)
	}
	t := strings.Join(r, ` `)

	// printer.Printf("DEBUG: txtEncode txt  encodedv=%v\n", t)
	return t
}
