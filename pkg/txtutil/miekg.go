package txtutil

import (
	"fmt"

	"codeberg.org/miekg/dns/pkg/pool"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
)

var builderPool = pool.NewBuilder()

// ZoneifyQuoted prints strings, each individually quoted and escaped, as used by Txt records.
// This is useful for spewing untrusted data into a zonefile, or URLs and other things that private types might want to use.
// Example: []string{"one", "tw o", "three"} outputs: `"one" "tw o" "three"`.
func ZoneifyQuoted(txts []string) string {
	rdtxt := dnsrdatav2.TXT{Txt: txts}
	return rdtxt.String()
}

// Zoneify is like ZoneifyQuoted, but omits the quotes when not needed. (Note:
// It might quote things that don't strictly need quoting, but it won't fail to
// quote things that do need quoting.)
// Example: []string{"one", "tw o", "three"} outputs: `one "tw o" three`.
func Zoneify(txts []string) string {
	sb := builderPool.Get()
	defer builderPool.Put(sb)

	for i, s := range txts {
		if i > 0 {
			sb.Grow(1)
			sb.WriteString(` `)
		}

		if isPlain(s) {
			sb.Grow(len(s))
			sb.WriteString(s)
		} else {
			q := ZoneifyStringQuoted(s)
			sb.Grow(2 + len(q))
			sb.WriteString(q)
		}
	}
	return sb.String()
}

// ZoneifyString is a convenience function for Zoneify when you have only one string.
func ZoneifyString(s string) string {
	if isPlain(s) {
		return s
	}
	return ZoneifyStringQuoted(s)
}
func ZoneifyStringQuoted(s string) string {
	rdtxt := dnsrdatav2.TXT{Txt: []string{s}}
	return rdtxt.String()
}

// ZoneifyManyAny is a convenience function for Zoneify when you have a []any and want a string.
func ZoneifyManyAny(args []any) string {
	n := make([]string, len(args))
	for i, arg := range args {
		n[i] = fmt.Sprint(arg)
	}
	return Zoneify(n)
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

		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			(r == '.') ||
			(r == '@') ||
			(r == '*') {
			continue
		}
		// it isn't any of those. We are NOT plain.
		return false
	}
	return true
}
