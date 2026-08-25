package models

import (
	"fmt"
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
	// Return true if name is the origin (should be "@") or ends in the origin.
	return name == origin || strings.HasSuffix(name, "."+origin)
}

func stutterError(rc *RecordConfig, domain string) error {
	label := rc.Name
	shortname := strings.TrimSuffix(label, "."+domain)
	return fmt.Errorf(
		`%s The target name "%s.%s." is an error (repeats the domain). Possible fixes: Replace %q with %q or %q or add DISABLE_REPEATED_DOMAIN_CHECK to this record to override`,
		rc.FilePos,
		label, domain,
		label,
		shortname,
		label+".",
	)
}
