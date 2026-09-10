package hostingde

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// A zone without an SOA record in dnsconfig.js must not have its contact
// address rewritten, so the placeholder has to carry an empty mailbox.
func TestPlaceholderSOAHasNoMailbox(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc, err := placeholderSOA(dc)
	if err != nil {
		t.Fatal(err)
	}

	if got := rc.AsSOA().Mbox; got != "" {
		t.Errorf("placeholder mailbox = %q, want empty; a non-empty one is turned into %s@example.com", got, got)
	}
}

func TestSoaMailToEmail(t *testing.T) {
	tests := []struct {
		name string
		mbox string
		want string
	}{
		{"0", "", ""},
		{"1", "hostmaster", "hostmaster@example.com"},
		{"2", "hostmaster.example.com.", "hostmaster@example.com"},
		// The host may be outside the zone.
		{"3", "eee.cloudflare.com.", "eee@cloudflare.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := soaMailToEmail(tt.mbox, "example.com"); got != tt.want {
				t.Errorf("soaMailToEmail(%v) = %v, want %v", tt.mbox, got, tt.want)
			}
		})
	}
}
