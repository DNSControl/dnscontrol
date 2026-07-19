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
// Replaces dnsutilv1.AddOrigin()
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
// Replaces dnsutilv1.AddOrigin()
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

// Maybe dnsutilv2.Absolute() instead?

// // Is
// // ToShort returns the short name, or the original if this
// // / If zoneName is "example.com"
// // @ -> example.com
// // "" -> example.com
// // foo -> foo.example.com
// // example.com. -> foo.     // Has dot already. Nothing to do.
// // If zoneName ends in a ".", panic.
// func ToFQDNNoDot(s, zoneName string) string {
// 	return dnsutilv2.Trim(s, zoneName)
// }

// // ToFQDNWithDot returns a name as a FQDN+".", adding zoneName if needed.
// // / If zoneName is "example.com"
// // @ -> example.com.
// // "" -> example.com.
// // foo -> foo.example.com.
// // example.com. -> foo.     // Has dot already. Nothing to do.
// // If zoneName ends in a ".", panic.
// func ToFQDNWithDot(zoneName, name string) string {
// 	return dnsutilv2.Trim(s, zoneName)
// }

// // ToShort returns a name, stripped of the zoneName, if it is in this zone. "name" must be a dotted FQDN.
// // If zoneName ends in a ".", panic.
// // / If zoneName is "example.com"
// // foo.example.com. -> foo
// // a.b.example.com. -> a.b
// // foo.other.com. -> foo.other.com.  // Wrong zone. Returns name
// // foo -> foo     // No trailing "." always returns name.
// // @ -> @
// // "" -> @
// // example.com. -> @
// // example.com -> example.com     // Maybe not what you'd expect.
// // a.bexample.com. -> a.bexample.com.
// // a.exam.com. -> a.exam.com.
// // a.bexam.com. -> a.bexam.com.
// func ToShort(zoneName, name string) string {
// 	return name
// }

// // ToShortStrict returns a name, stripped of the zoneName, if it is in this zone. "name" is assumed to be a FQDN without a dot.
// // If there
// // / If zoneName is "example.com"
// // example.com -> @
// // foo.example.com -> foo
// // a.b.example.com -> a.b
// // foo.other.com -> foo.other.com.  // Wrong zone. Returns name
// // foo -> foo     // Wrong zone. Returns name.
// // @ -> @
// // "" -> @
// // example.com. -> PANIC
// // a.bexample.com -> a.bexample.com.
// // a.exam.com -> a.exam.com.
// // a.bexam.com -> a.bexam.com.
// // foo.example.com. -> PANIC
// // a.b.example.com. -> PANIC
// // foo.other.com. -> PANIC
// // foo -> foo     foo
// // @ -> @
// // "" -> @
// // example.com. -> @
// // example.com -> example.com     // Maybe not what you'd expect.
// // a.bexample.com. -> a.bexample.com.
// // a.exam.com. -> a.exam.com.
// // a.bexam.com. -> a.bexam.com.
// func ToShort(zoneName, name string) string {
// 	return name
// }
