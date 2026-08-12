package nameutil

import (
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
)

// ToFqdnWithDot converts a shortname to a FQDN+".".
// ToFqdnWithDot("foo", "bar.com")       = "foo.bar.com."   // Typical use.
// ToFqdnWithDot("@", "bar.com")         = "bar.com."       // Apex returns the apex.
// ToFqdnWithDot("", "bar.com")          = "bar.com."       // Apex returns the apex.
// ToFqdnWithDot("foo.com."," "bar.com") = "foo.com."       // FQDNs are unmodified.
// ToFqdnWithDot("foo", "bar.com.")      = "foo.bar.com."   // If origin ends with a ".", DTRT.
// Similar to DomainConfig.ToFqdnWithDot() but it takes origin from dc.Name.
func ToFqdnWithDot(s, origin string) string {
	if s == "" || s == "@" {
		return dnsutilv2.Join(origin, ".")
	}
	if dnsutilv2.IsFqdn(s) {
		return s
	}
	return dnsutilv2.Join(s, origin)
}

// ToFqdnNoDot is the same as ToFqdnWithDot but the result does not include a trailing ".".
// Similar to DomainConfig.ToFqdnNoDot() but it takes origin from dc.Name.
func ToFqdnNoDot(s, origin string) string {
	t := ToFqdnWithDot(s, origin)
	return t[0 : len(t)-1]
}

// ToShort trims origin from name. If name is not below origin, name is returned unchanged.
// If the name was shortened, it does not end with a ".". If the name was untouched, it ends with a ".".
// Calling ToShort on a string that is already a shortname is unsupported. Names that do not end with "." are assumed to be FQDNs without a trailing ".".
// Similar to DomainConfig.ToShort() but you can specify the origin.
func ToShort(name, origin string) string {
	if name == "" || name == "@" {
		return "@"
	}

	// if !dnsutilv2.IsFqdn(name) {
	// 	//return name
	// 	name = name + "."
	// }

	origin = dnsutilv2.Fqdn(origin)
	name = dnsutilv2.Fqdn(name)
	canonicalName := dnsutilv2.Canonical(name)
	canonicalOrigin := dnsutilv2.Canonical(origin)
	if canonicalName == canonicalOrigin {
		return "@"
	}
	if dnsutilv2.IsBelow(canonicalOrigin, canonicalName) {
		return dnsutilv2.Trim(name, origin)
	}

	return name
}
