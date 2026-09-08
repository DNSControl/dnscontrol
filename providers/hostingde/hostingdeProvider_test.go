package hostingde

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// A zone without an SOA record in dnsconfig.js must not have its contact
// address rewritten, so the placeholder has to carry an empty mailbox.
func TestPlaceholderSOAHasNoMailbox(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	if got := placeholderSOA(dc).AsSOA().Mbox; got != "" {
		t.Errorf("placeholder mailbox = %q, want empty; a non-empty one is turned into %s@example.com", got, got)
	}
}
