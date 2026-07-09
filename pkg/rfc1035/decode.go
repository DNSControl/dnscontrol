package rfc1035

// Decode an RFC1035-encoded TXT description.

import (
	"bytes"
	"fmt"
)

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

	// Processing the three digits of a \DDD
	StateDDD1
	StateDDD2
	StateDDD3
	StateDDD1Quote
	StateDDD2Quote
	StateDDD3Quote

	// StateWantSpace indicates parser expects a space (the previous token was a closing quote).
	StateWantSpace
)

// Decode decodes a TXT descriptoin as Tom inteprets RFC1035, which is a very loose interpretation such that all strings (even with binary data) can be decoded.
func Decode(s string) []string {
	// Parse according to RFC1035 zonefile specifications.
	// "foo"  -> one string: `foo``
	// "foo" "bar"  -> two strings: `foo` and `bar`
	// quotes and backslashes are escaped using \" and \\.
	// \x is that (printable) char, \DDD is a decimal ascii code.
	/*

		BNF:
			txttarget := `""`` | item | item ` item*
			item := quoteditem | unquoteditem
			quoteditem := quote innertxt quote
			:= `"`
			innertxt := (escaped | printable )*
			escaped := `\x` | `\x` | `\ddd`   # where "x" is any printable and "ddd" is three digits.
			printable := (printable ASCII chars)
			unquoteditem := (printable ASCII chars but not `"` nor ' ')

	*/

	var value int               // A \ddd value as its being constructed.
	var returnToThisState State // When processing a \ddd, which state to return to.
	var result []string         // The result as we know it.

	b := &bytes.Buffer{}
	state := StateStart
	for i, c := range []byte(s) {
		// printer.Printf("DEBUG: state=%v rune=%v\n", state, string(c))

		switch state {
		case StateStart:
			switch c {
			case ' ': // skip whitespace
			case '"': // quoted text segment has begun
				state = StateQuoted
			case '\\': // backslash start
				returnToThisState = StateUnquoted

				if IsRemaining(s, i, 3) && next3AreDigits(s, i) {
					state = StateDDD1
				} else if IsRemaining(s, i, 1) { // Is there at least 1 more bytes?
					state = StateBackslash
				} else {
					state = returnToThisState
					b.WriteByte(c) // Error. Just move along.
				}
			default: // unquoted text segment has begun
				state = StateUnquoted
				b.WriteByte(c)
			}

		case StateUnquoted:
			switch c {
			case ' ': // space ends a segment
				state = StateStart
				result = append(result, b.String())
				b = &bytes.Buffer{}
			case '\\': // backslash start
				returnToThisState = StateUnquoted

				if IsRemaining(s, i, 3) && next3AreDigits(s, i) {
					state = StateDDD1
				} else if IsRemaining(s, i, 1) { // Is there at least 1 more bytes?
					state = StateBackslash
				} else {
					state = returnToThisState
					b.WriteByte(c) // Error. Just move along.
				}
			default:
				b.WriteByte(c)
			}

		case StateQuoted:
			switch c {
			case '\\':
				returnToThisState = StateQuoted

				if IsRemaining(s, i, 3) && next3AreDigits(s, i) {
					state = StateDDD1
				} else if IsRemaining(s, i, 1) { // Is there at least 1 more bytes?
					state = StateBackslash
				} else {
					state = returnToThisState
					b.WriteByte(c) // Error. Just move along.
				}
			case byte('"'):
				state = StateWantSpace
				result = append(result, b.String())
				b = &bytes.Buffer{}
			default:
				b.WriteByte(c)
			}

		case StateDDD1:
			state = StateDDD2
			value = int(c - '0')
		case StateDDD2:
			state = StateDDD3
			value = (value * 10) + int(c-'0')
		case StateDDD3:
			state = returnToThisState
			value = (value * 10) + int(c-'0')
			if value < 0 || value > 255 {
				b.WriteString(fmt.Sprintf("\\%0d", value))
			} else {
				b.WriteByte(byte(value))
			}

		case StateBackslash:
			state = returnToThisState
			b.WriteByte(c)

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
				state = StateUnquoted
			}
		}
	}

	// What state were we in when we hit EOF?
	switch state {
	case StateStart, StateWantSpace:
		// no-op
	case StateUnquoted, StateQuoted, StateBackslash:
		result = append(result, b.String())
	case StateDDD1, StateDDD2, StateDDD3:
		panic("Should not happen")
	}

	return result
}

func IsRemaining(s string, i, r int) bool {
	//fmt.Printf("DEBUG: isRemaining(%s, %d, %d) = %d (%t)\n", s, i, r, (len(s) - 1 - i), ((len(s) - 1 - i) > r))
	return (len(s) - 1 - i) >= r
}

func next3AreDigits(s string, i int) bool {
	return isDigit(s[i+1]) && isDigit(s[i+2]) && isDigit(s[i+3])
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
