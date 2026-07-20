package name

import (
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
)

// ToFqdnWithDot converts a shortname to a FQDN+".".
// ToFqdnWithDot("foo", "bar.com")     = "foo.bar.com."   // Typical use.
// ToFqdnWithDot("@", "bar.com")       = "bar.com."       // Apex returns the apex.
// ToFqdnWithDot("", "bar.com")        = "bar.com."       // Apex returns the apex.
// ToFqdnWithDot("foo.com. "bar.com")  = "foo.com."       // FQDNs are unmodified.
// ToFqdnWithDot("foo", "bar.com.")    = "foo.bar.com."   // If origin ends with a ".", DTRT.
// Replaces dnsutilv1.AddOrigin().
func ToFqdnWithDot(s, origin string) string {
	if s == "" || s == "@" {
		return dnsutilv2.Join(origin, ".")
	}
	if dnsutilv2.IsFqdn(s) {
		return s
	}
	return dnsutilv2.Join(s, origin)
}

// ToFqdnNoDot is the same as ToFqdnWithDot but the trailing "." is removed.
// Replaces dnsutilv1.AddOrigin().
func ToFqdnNoDot(s, origin string) string {
	t := ToFqdnWithDot(s, origin)
	return t[0 : len(t)-1]
}

func ToShort(s, origin string) string {
	if s == "" || s == "@" {
		return "@"
	}

	if !dnsutilv2.IsFqdn(s) {
		return s
	}

	origin = dnsutilv2.Fqdn(origin)
	canonicalName := dnsutilv2.Canonical(s)
	canonicalOrigin := dnsutilv2.Canonical(origin)
	if canonicalName == canonicalOrigin {
		return "@"
	}
	if dnsutilv2.IsBelow(canonicalOrigin, canonicalName) {
		return dnsutilv2.Trim(s, origin)
	}

	return s
}
