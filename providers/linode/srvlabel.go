package linode

import (
	"fmt"
	"regexp"
	"strings"
)

// srvLabelRegexp matches a valid SRV record label: "_service._protocol",
// optionally followed by a subdomain (e.g. "_smtp._tcp.sub.domain").  This is
// used both by AuditRecords to validate labels and by toReq to extract Service
// and Protocol from the labels.
var srvLabelRegexp = regexp.MustCompile(`^_([[:alnum:]-_]+)\._([[:alnum:]_])`)

func validateSrvLabel(label string) error {
	_, _, err := validateSrvLabelHelper(label)
	return err
}

func extractSrvLabelValues(label string) (string, string, error) {
	service, protocol, err := validateSrvLabelHelper(label)
	if err != nil {
		return "", "", err
	}

	/*
		We make no attempt at validating service name or protocol beyond
		that srvLabelRegexp allows.

		The user will get a reject during "push", which isn't optimal, but we can't
		attempt to emulate Linode's algorithm, which could change without notice.

		Attempts to document Linode's algorithm are here:
		https://github.com/DNSControl/dnscontrol/issues/4812#issuecomment-5444147757

	*/

	return service[1:], protocol[1:], nil
}

func validateSrvLabelHelper(label string) (string, string, error) {
	parts := strings.SplitN(label, ".", 3)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid label %q: fewer than 2 parts", label)
	}
	front := label[0 : len(parts[0])+1+len(parts[1])]
	if !srvLabelRegexp.MatchString(front) {
		return "", "", fmt.Errorf("invalid label %q", label)
	}
	return parts[0], parts[1], nil
}
