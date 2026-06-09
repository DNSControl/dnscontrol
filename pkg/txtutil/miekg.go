package txtutil

import (
	"fmt"
	"strings"

	"codeberg.org/miekg/dns/pkg/pool"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil/ddd"
)

var builderPool = pool.NewBuilder()

// ZoneifyQuoted prints strings, each individually quoted and escaped, as used by Txt records.
// However, this is also useful for spewing untrusted data into a zonefile, or URLs and other things that private types might want to use.
// Example: []string{"one", "two", "three"} outputs: `"one" "two" "three"`
// TODO: Request to upstream this and make it a public function in miekg/dns, and then remove this code from dnscontrol.
// TODO: Harden this so that it works with all possible strings, including backslashes, binary data, etc.
func ZoneifyQuoted(txt []string) string {
	sb := builderPool.Get()
	defer builderPool.Put(sb)

	for i, s := range txt {
		sb.Grow(3 + len(s))
		if i > 0 {
			sb.WriteString(` "`)
		} else {
			sb.WriteByte('"')
		}
		for j := 0; j < len(s); {
			b, n := ddd.Next(s, j)
			if n == 0 {
				break
			}
			writeTxtByte(&sb, b)
			j += n
		}
		sb.WriteByte('"')
	}
	return sb.String()
}

// Zoneify is like ZoneifyQuoted, but omits the quotes when not needed. (Note:
// It might quote things that don't strictly need quoting, but it won't fail to
// quote things that do need quoting.)
// Example: []string{"one", "two", "three"} outputs: `"one" "two" "three"`.
func Zoneify(txt []string) string {
	sb := builderPool.Get()
	defer builderPool.Put(sb)

	for i, s := range txt {
		if i > 0 {
			sb.Grow(1)
			sb.WriteString(` `)
		}

		if isPlain(s) {
			sb.Grow(len(s))
			sb.WriteString(s)
		} else {
			sb.Grow(2 + len(s))
			sb.WriteByte('"')
			for j := 0; j < len(s); {
				b, n := ddd.Next(s, j)
				if n == 0 {
					break
				}
				writeTxtByte(&sb, b)
				j += n
			}
			sb.WriteByte('"')
		}
	}
	return sb.String()
}

// ZoneifyString is a convenience function for Zoneify when you have only one string.
func ZoneifyString(s string) string {
	return Zoneify([]string{s})
}

// ZoneifyManyAny is a convenience function for Zoneify when you have a []any and want a string.
func ZoneifyManyAny(args []any) string {
	n := make([]string, len(args))
	for i, arg := range args {
		n[i] = fmt.Sprintf("%s", arg)
	}
	return Zoneify(n)
}

func writeTxtByte(sb *strings.Builder, b byte) {
	switch {
	case b == '"' || b == '\\':
		sb.WriteByte('\\')
		sb.WriteByte(b)
	case b < ' ' || b > '~':
		sb.WriteString(ddd.Escape(b))
	default:
		sb.WriteByte(b)
	}
}

// isPlain returns true if the string doesn't need to be quoted.
// It errs on the side of caution, including only A-Z, a-z, 0-9, and ".", "@", and "*".
// TODO: Optimize this code. Maybe use strings.ContainsAny() ?
func isPlain(s string) bool {
	if s == "" {
		return false // Null string always requires quotes.
	}
	for _, r := range s {
		// continue if we are safe: a-z A-Z 0-9 . @ *

		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '.' {
			continue
		}
		if r == '@' {
			continue
		}
		if r == '*' {
			continue
		}
		return false
	}
	return true
}
