package transform

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/pkg/rfc4183"
)

// ReverseDomainName turns a CIDR block into a reversed (in-addr) name.
// For cases not covered by RFC2317, implement RFC4183
// The host bits must all be zeros.
func ReverseDomainName(cidr string) (string, error) {
	if rfc4183.IsRFC4183Mode() {
		return rfc4183.ReverseDomainName(cidr)
	}

	// Mask missing? Add it.
	if !strings.Contains(cidr, "/") {
		a, err := netip.ParseAddr(cidr)
		if err != nil {
			return "", fmt.Errorf("not an IP address: %w", err)
		}
		if a.Is4() {
			cidr = cidr + "/32"
		} else {
			cidr = cidr + "/128"
		}
	}

	// Parse the CIDR.
	p, err := netip.ParsePrefix(cidr)
	if err != nil {
		return "", fmt.Errorf("not a CIDR block: %w", err)
	}
	bits := p.Bits()

	if p.Masked() != p {
		return "", fmt.Errorf("CIDR %v has 1 bits beyond the mask", cidr)
	}

	// Cases where RFC4183 is the same as RFC2317:
	// IPV6, /0 - /24, /32
	if strings.Contains(cidr, ":") || bits <= 24 || bits == 32 {
		// There is no p.Is6() so we test for ":" as a workaround.
		return rfc4183.ReverseDomainName(cidr)
	}

	// Record that the change to --revmode default will affect this configuration
	rfc4183.NeedsWarning()

	// Handle IPv4 "Classless in-addr.arpa delegation" RFC2317:
	// if bits >= 25 && bits < 32 {
	// first address / netmask . Class-b-arpa.

	ip := p.Addr().AsSlice()
	return fmt.Sprintf("%d/%d.%d.%d.%d.in-addr.arpa",
		ip[3], bits, ip[2], ip[1], ip[0]), nil
}
