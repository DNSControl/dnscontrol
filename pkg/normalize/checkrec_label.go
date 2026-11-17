package normalize

import (
	"fmt"
	"slices"
	"strings"
)

func checkLabel(label string, rType string, domain string) error {
	if label == "@" {
		return nil
	}

	// Underscores are permitted in labels, but we print a warning unless they
	// are used in a way we consider typical.  Yes, we're opinionated here.

	// Don't warn for certain rtypes:
	if slices.Contains([]string{"SRV", "TLSA", "TXT", "LUA"}, rType) {
		return nil
	}
	// Don't warn for records that start with _
	// See https://github.com/DNSControl/dnscontrol/issues/829
	if strings.HasPrefix(label, "_") || strings.Contains(label, "._") || strings.HasPrefix(label, "sql-") {
		return nil
	}

	// Otherwise, warn.
	if strings.ContainsRune(label, '_') {
		return Warning{fmt.Errorf("label %s.%s contains \"_\" (can't be used in a URL)", label, domain)}
	}

	return nil
}
