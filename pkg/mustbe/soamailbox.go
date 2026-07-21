package mustbe

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/pkg/domaintags"
)

// SoaMailbox format a SOA Mailbox string, such as tal.example.com (The "@" in
// the email address is written as an "." because nothing in DNS is simple).
// This function also...
// * Turns normal email addresses (tal@example.com) into SOA Mailbox strings (tal.example.com).
// * Turns Unicode into PunyCode (just in the hostname part)
// * Runs ToLower.
// * Does NOT add an origin if this is a shortname. We can't know if it is a shortname because it is dotless.)
func SoaMailbox(arg any) string {
	var name string
	switch v := arg.(type) {
	case string:
		name = v
	case int:
		name = fmt.Sprintf("%d", arg)
	default:
		name = fmt.Sprintf("%v", arg)
	}

	// Turn email addresses into SOAMailbox format:
	name = strings.ReplaceAll(name, "@", ".")
	name = strings.ToLower(name)

	if strings.Count(name, ".") == 0 {
		// Techinically we should reject this but... they're digging their own grave.
		return name
	}

	// Normalize it (even though it isn't a hostname,
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 0 {
		return "SHOULD_NOT_HAPPEN."
	}
	if len(parts) == 1 {
		parts = append(parts, "")
	}
	username, hostname := parts[0], parts[1]

	hostname = domaintags.EfficientToASCII(hostname)

	return username + "." + hostname
}
