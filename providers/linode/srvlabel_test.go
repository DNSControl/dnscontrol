package linode

import (
	"testing"
)

// invalidSrvLabels is a list of invalid labels, for use in tests.
var invalidSrvLabels = []string{
	"_foo&._tcp", // '&' is not a word character or hyphen
	"_smtp",      // missing the protocol token
	"notasrv",    // missing the leading underscore
}

// srvLabelTests is a list of valid labels, with expected service and protocol extraction.
var srvLabelTests = []struct {
	label    string
	service  string
	protocol string
}{
	{label: "_muble._udp-lite", service: "muble", protocol: "udp-lite"},
	{label: "_sip._udp", service: "sip", protocol: "udp"},
	{label: "_smtp._tcp", service: "smtp", protocol: "tcp"},
	{label: "_tcp._smtp", service: "tcp", protocol: "smtp"},
	{label: "_xmpp-server._tcp", service: "xmpp-server", protocol: "tcp"},
	// Underscores are permitted within the service/protocol tokens.
	{label: "_foo_bar._tcp", service: "foo_bar", protocol: "tcp"},
	// Permitted, but Linode will correct, which causes an update loop. We're ok
	// with that because anything else would require us to emulate the Linode
	// algorithm exactly, which is impossible because we don't have their source
	// code and we can't magically stay in sync with future changes they may
	// make. We fix this kind of thing by documenting it.
	{label: "__smtp._tcp", service: "_smtp", protocol: "tcp"},
	{label: "_sm_tp._tcp", service: "sm_tp", protocol: "tcp"},
	{label: "_smtp_._tcp", service: "smtp_", protocol: "tcp"},
	{label: "_smtp.__tcp", service: "smtp", protocol: "_tcp"},
}

func TestSRVLabel(t *testing.T) {

	for _, label := range invalidSrvLabels {
		t.Run("invalid/"+label, func(t *testing.T) {
			err := validateSrvLabel(label)
			if err == nil {
				t.Errorf("expected %q to be an invalid SRV label, but it was accepted", label)
			}
		})
	}

	for _, tc := range srvLabelTests {
		label := tc.label
		t.Run("valid/"+label, func(t *testing.T) {
			err := validateSrvLabel(label)
			if err != nil {
				t.Errorf("expected %q to be a valid SRV label, but it was rejected: %v", label, err)
			}
		})
	}
}

// TestExtractSrvParts verifies that srvLabelRegexp (the same regexp that
// validates the label) extracts the service and protocol properly.
func TestExtractSrvParts(t *testing.T) {
	for _, tc := range srvLabelTests {
		for _, suffix := range []string{"", ".sub", ".sub.dom"} {
			label := tc.label + suffix
			t.Run(label, func(t *testing.T) {
				service, protocol, err := extractSrvLabelValues(label)
				if err != nil {
					t.Fatalf("expected %q to be validated but it failed", tc.label)
				}
				if service != tc.service {
					t.Errorf("service = %q, want %q", service, tc.service)
				}
				if protocol != tc.protocol {
					t.Errorf("protocol = %q, want %q", protocol, tc.protocol)
				}
			})
		}
	}
}
