package transform

import (
	"fmt"
	"net" // Permitted import for using legacy net.CIDRMask and related functions.
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

// PtrNameMagic implements the PTR magic.
func PtrNameMagic(name, domain string) (string, error) {
	// Implement the PTR name magic.  If the name is a properly formed
	// IPv4 or IPv6 address, we replace it with the right string (i.e
	// reverse it and truncate it).

	// If the name is already in-addr.arpa or ip6.arpa,
	// make sure the domain matches.
	if strings.HasSuffix(name, ".in-addr.arpa.") || strings.HasSuffix(name, ".ip6.arpa.") {
		if before, ok := strings.CutSuffix(name, "."+domain+"."); ok {
			return before, nil
		}
		return name, fmt.Errorf("PTR record %v in wrong domain (%v)", name, domain)
	}

	// If the domain is .arpa, we do magic.
	if strings.HasSuffix(domain, ".in-addr.arpa") {
		return ipv4magic(name, domain)
	} else if strings.HasSuffix(domain, ".ip6.arpa") {
		return ipv6magic(name, domain)
	} else {
		return name, nil
	}
}

func ipv4magic(name, domain string) (string, error) {
	// Not a valid IPv4 address. Leave it alone.
	ip, err := netip.ParseAddr(name)
	if err != nil || !ip.IsValid() || !ip.Is4() || !strings.Contains(name, ".") {
		return name, nil
	}

	// Reverse it.
	rev, err := ReverseDomainName(ip.String() + "/32")
	if err != nil {
		return name, err
	}
	result := strings.TrimSuffix(rev, "."+domain)

	// Are we in the right domain?
	if strings.HasSuffix(rev, "."+domain) {
		return result, nil
	}
	octets := ipMatchesClasslessDomain(ip, domain)
	if octets > 0 {
		return strings.Join(strings.SplitN(rev, ".", octets+1)[0:octets], "."), nil
	}

	return "", fmt.Errorf("PTR record %v in wrong IPv4 domain (%v)", name, domain)
}

var isRfc2317Format1 = regexp.MustCompile(`(\d{1,3})/(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.in-addr\.arpa$`)
var isRfc4183Format1 = regexp.MustCompile(`(\d{1,3})-(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.in-addr\.arpa$`)
var isRfc4183Format2 = regexp.MustCompile(`(\d{1,3})-(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.in-addr\.arpa$`)
var isRfc4183Format3 = regexp.MustCompile(`(\d{1,3})-(\d{1,3})\.(\d{1,3})\.in-addr\.arpa$`)

// ipMatchesClasslessDomain returns true if ip is appropriate for domain.
// domain is a reverse DNS lookup zone (in-addr.arpa) as described in RFC2317.
func ipMatchesClasslessDomain(ipAddress netip.Addr, domain string) int {

	// Converting netip.Addr to net.IP so we can use the legacy code, which is troublesome to translate.
	// get Addr as []byte
	s := ipAddress.AsSlice()
	// convert bytes slice to net.IP type (it works as net.IP underlying type is also []byte so net.IP and []byte are identical)
	ip := net.IP(s)

	// The unofficial but preferred format in RFC2317:
	m := isRfc2317Format1.FindStringSubmatch(domain)
	if m != nil {
		// IP:          Domain:
		// 172.20.18.27 128/27.18.20.172.in-addr.arpa
		// A   B  C  D  F   M  X  Y  Z
		// The following should be true:
		//   A==Z, B==Y, C==X.
		//   If you mask ip by M, the last octet should be F.
		ii := ip.To4()
		a, b, c, _ := ii[0], ii[1], ii[2], ii[3]
		f, m, x, y, z := atob(m[1]), atob(m[2]), atob(m[3]), atob(m[4]), atob(m[5])
		masked := ip.Mask(net.CIDRMask(int(m), 32))
		if a == z && b == y && c == x && masked.Equal(net.IPv4(a, b, c, f)) {
			return 1
		}
	}

	// The format in RFC4183 for /25 to /32:
	m = isRfc4183Format1.FindStringSubmatch(domain)
	if m != nil {
		// IP:          Domain:
		// 172.20.18.27 128-27.18.20.172.in-addr.arpa
		// A   B  C  D  F   M  X  Y  Z
		// The following should be true:
		//   A==Z, B==Y, C==X.
		//   If you mask ip by M, the last octet should be F.
		ii := ip.To4()
		a, b, c, _ := ii[0], ii[1], ii[2], ii[3]
		f, m, x, y, z := atob(m[1]), atob(m[2]), atob(m[3]), atob(m[4]), atob(m[5])
		masked := ip.Mask(net.CIDRMask(int(m), 32))
		if a == z && b == y && c == x && masked.Equal(net.IPv4(a, b, c, f)) {
			return 1
		}
	}

	// The format in RFC4183 for /17 to /23:
	m = isRfc4183Format2.FindStringSubmatch(domain)
	if m != nil {
		// IP:          Domain:
		// 172.20.18.27 128-27.20.172.in-addr.arpa
		// A   B  C  D  F   M  Y  Z
		// The following should be true:
		//   A==Z, B==Y,
		//   If you mask ip by M, the second last octet should be F.
		ii := ip.To4()
		a, b, _, _ := ii[0], ii[1], ii[2], ii[3]
		f, m, y, z := atob(m[1]), atob(m[2]), atob(m[3]), atob(m[4])
		masked := ip.Mask(net.CIDRMask(int(m), 32))
		if a == z && b == y && masked.Equal(net.IPv4(a, b, f, 0)) {
			return 2
		}
	}

	// The format in RFC4183 for /9 to /15:
	m = isRfc4183Format3.FindStringSubmatch(domain)
	if m != nil {
		// IP:          Domain:
		// 172.20.18.27 128-27.172.in-addr.arpa
		// A   B  C  D  F   M  Z
		// The following should be true:
		//   A==Z,
		//   If you mask ip by M, the third last octet should be F.
		ii := ip.To4()
		a, _, _, _ := ii[0], ii[1], ii[2], ii[3]
		f, m, z := atob(m[1]), atob(m[2]), atob(m[3])
		masked := ip.Mask(net.CIDRMask(int(m), 32))
		if a == z && masked.Equal(net.IPv4(a, f, 0, 0)) {
			return 3
		}
	}

	// To extend this to include other formats, add them here.

	return 0
}

// atob converts a to a byte value or panics.
func atob(s string) uint8 {
	if i, err := strconv.ParseUint(s, 10, 8); err == nil {
		return byte(i)
	}
	panic(fmt.Sprintf("%v is not a byte", s))
}

func ipv6magic(name, domain string) (string, error) {
	// Not a valid IPv6 address. Leave it alone.
	ip := net.ParseIP(name)
	if ip == nil || len(ip) != 16 || !strings.Contains(name, ":") {
		return name, nil
	}

	// Reverse it.
	rev, err := ReverseDomainName(ip.String() + "/128")
	if err != nil {
		return name, err
	}
	if !strings.HasSuffix(rev, "."+domain) {
		err = fmt.Errorf("PTR record %v in wrong IPv6 domain (%v)", name, domain)
	}
	return strings.TrimSuffix(rev, "."+domain), err
}
