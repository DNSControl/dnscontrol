package rfc4183

import "errors"

// reverseIPv6 returns the ip6.arpa string suitable for reverse DNS lookups.
func reverseIPv6(ip []byte, maskbits int) (arpa string, err error) {
	// Must be IPv6
	if len(ip) != 16 {
		return "", errors.New("not IPv6")
	}

	buf := []byte("x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.x.ip6.arpa")
	// Poke hex digits into the template
	pos := 128/4*2 - 2 // Position of the last "x"
	for _, v := range ip {
		buf[pos] = hexDigit[v>>4]
		buf[pos-2] = hexDigit[v&0xF]
		pos = pos - 4
	}
	// Return only the parts without x's
	return string(buf[(128-maskbits)/4*2:]), nil
}

const hexDigit = "0123456789abcdef"
