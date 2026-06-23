package models

import (
	"strings"
)

// isReverseZone reports whether zoneName is an IPv4 or IPv6 reverse-DNS zone.
func isReverseZone(zoneName string) bool {
	return strings.HasSuffix(zoneName, ".in-addr.arpa") || strings.HasSuffix(zoneName, ".ip6.arpa")
}

func doesStutter(name, origin string) bool {
	// TODO(tlim): MAYBE: Never return true if last char is "."?
	// TODO(tlim): Panic if called with name == ""?
	if name == "@" {
		return false
	}
	if name == origin || strings.HasSuffix(name, "."+origin) {
		return true
	}
	return false
}
