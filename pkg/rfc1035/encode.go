package rfc1035

// Encode TXT records in an RFC1035-encoded string.

import (
	"bytes"
	"fmt"
)

// Encode encodes a TXT record into an RFC1035-encoded string as Tom inteprets RFC1035, which is a very loose interpretation such that all strings (even with binary data) can be encoded.
func Encode(txts []string) string {
	if len(txts) == 0 {
		return `""`
	}

	b := &bytes.Buffer{}
	for i, txt := range txts {
		if i > 0 {
			b.WriteByte(' ')
		}

		if len(txts) == 1 && isPlainString(txt) {
			escape(b, txt)
		} else {
			b.WriteByte('"')
			escape(b, txt)
			b.WriteByte('"')
		}
	}
	return b.String()
}

// escape writes to b the string s, adding \DDD and \x escapes as needed.
func escape(b *bytes.Buffer, s string) {
	for _, c := range []byte(s) {
		if isPlainByte(c) {
			b.WriteByte(c)
		} else if c == ' ' {
			b.WriteByte(c)
		} else if c < 32 || c == 127 {
			fmt.Fprintf(b, `\%03d`, c)
		} else {
			b.WriteByte('\\')
			b.WriteByte(c)
		}
	}

}

// isPlainString returns true if the string doesn't need to be quoted.
// It errs on the side of caution, including only A-Z, a-z, 0-9, and ".", and "*".
// "@" is plain but strings that contain an "@" are not.
// TODO: Optimize this code. Maybe use strings.ContainsAny() ?
func isPlainString(s string) bool {
	if s == "" {
		return false // Null string always requires quotes.
	}
	if s == "@" {
		return false
	}

	for _, r := range []byte(s) {
		if !isPlainByte(r) {
			return false
		}
	}
	return true
}

func isPlainByte(c byte) bool {
	return ((c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		(c == '.') ||
		(c == '*'))
}
