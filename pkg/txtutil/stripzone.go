package txtutil

import (
	"strings"

	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
)

func StripZone(label, zone string) string {
	// Special cases.
	if label == "" || label == "@" {
		return "@"
	}
	if label == "*" {
		return label
	}

	// Normalize to make future logic easier.
	nlabel := strings.TrimSuffix(label, ".")
	nzone := zone
	if before, ok := strings.CutSuffix(zone, "."); ok {
		nzone = before
	}
	nzone = strings.TrimPrefix(nzone, ".")

	if before, found := strings.CutSuffix(nlabel, "."+nzone); found {
		return before
	}
	if nlabel == nzone {
		return "@"
	}
	return label
}

func Extend(host, zone string) string {
	if zone == "" {
		// TODO(tlim): Issue a warning? This is a "flag" that means
		// "just return the host". It is generally used in legacy code
		// that doesn't know the origin/zone name. Adding a warning
		// here would help find legacy code that needs to be replaced.
		return host
	}
	if dnsutilv2.IsFqdn(host) {
		return host
	}
	return dnsutilv2.Absolute(host, zone)
}
