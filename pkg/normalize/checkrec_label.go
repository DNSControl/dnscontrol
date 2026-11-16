package normalize

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

func errorRepeat(label, domain string) string {
	shortname := strings.TrimSuffix(label, "."+domain)
	return fmt.Sprintf(
		`The name "%s.%s." is an error (repeats the domain). Maybe instead of "%s" you intended "%s"? If not add DISABLE_REPEATED_DOMAIN_CHECK to this record to permit this as-is.`,
		label, domain,
		label,
		shortname,
	)
}

func checkLabel(label string, rType string, domain string, meta map[string]string) error {
	if label == "@" {
		return nil
	}
	if label == "" {
		return fmt.Errorf("empty %s label (\"\") in %s. Did you mean \"@\" instead?", rType, domain)
	}
	if label[len(label)-1] == '.' {
		return fmt.Errorf("label %s.%s ends with a (.)", label, domain)
	}
	if label == domain || strings.HasSuffix(label, "."+domain) {
		if m := meta["skip_fqdn_check"]; m != "true" {
			return errors.New(errorRepeat(label, domain))
		}
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
