package linode

import (
	"testing"
)

func TestSRVLabel(t *testing.T) {
	valid := []string{
		"_smtp._tcp.subdomain",
		"_xmpp-server._tcp",
		"_muble._udp-lite",
		"_tcp._smtp.sub.domain",
		"_tcp._smtp.sub-domain",
	}
	invalid := []string{
		"__smtp._tcp",
		"_sm_tp._tcp",
		"_smtp_._tcp",
		"_smtp.__tcp",
		"_foo&._tcp",
	}

	for _, label := range valid {
		t.Run("valid/"+label, func(t *testing.T) {
			if !srvLabelRegexp.MatchString(label) {
				t.Errorf("expected %q to be a valid SRV label, but it was rejected", label)
			}
		})
	}

	for _, label := range invalid {
		t.Run("invalid/"+label, func(t *testing.T) {
			if srvLabelRegexp.MatchString(label) {
				t.Errorf("expected %q to be an invalid SRV label, but it was accepted", label)
			}
		})
	}
}

// TestExtractSrvParts verifies that srvLabelRegexp — the same regexp that
// validates the label — extracts the service and protocol from its named
// capture groups. A trailing subdomain is allowed but ignored by the
// extraction; interior underscores, a single label, or junk do not match.
func TestExtractSrvParts(t *testing.T) {
	tests := []struct {
		label    string
		service  string
		protocol string
		match    bool
	}{
		{label: "_smtp._tcp", service: "smtp", protocol: "tcp", match: true},
		{label: "_sip._udp", service: "sip", protocol: "udp", match: true},
		{label: "_xmpp-server._tcp", service: "xmpp-server", protocol: "tcp", match: true},
		{label: "_muble._udp-lite", service: "muble", protocol: "udp-lite", match: true},
		// A subdomain is permitted; extraction still yields service/protocol.
		{label: "_smtp._tcp.subdomain", service: "smtp", protocol: "tcp", match: true},
		// No match: an interior underscore, a single label, or junk.
		{label: "_foo_bar._tcp", match: false},
		{label: "_smtp", match: false},
		{label: "notasrv", match: false},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			result := srvLabelRegexp.FindStringSubmatch(tc.label)
			if !tc.match {
				if result != nil {
					t.Errorf("expected %q not to match, but got %v", tc.label, result)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected %q to match, but it did not", tc.label)
			}
			if got := result[1]; got != tc.service {
				t.Errorf("service = %q, want %q", got, tc.service)
			}
			if got := result[2]; got != tc.protocol {
				t.Errorf("protocol = %q, want %q", got, tc.protocol)
			}
		})
	}
}
