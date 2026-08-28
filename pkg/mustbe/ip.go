package mustbe

import (
	"fmt"
	"net"
	"net/netip"
)

func IPv4(a any) (netip.Addr, error) {
	switch v := a.(type) {
	case float64:
		// IP addresses expressed as a number (e.g. via the IP() helper in
		// dnsconfig.js) range from 0 to 4294967295 (2^32-1)
		i := uint32(v)
		x := netip.AddrFrom4([4]byte{
			byte(i >> 24), // NB(tlim): byte() silently truncates the upper bits.
			byte(i >> 16),
			byte(i >> 8),
			byte(i),
		})
		return x, nil
	case string:
		a, err := netip.ParseAddr(v)
		if err != nil || !a.Is4() {
			return netip.Addr{}, fmt.Errorf("not a valid IPv4 address: %v", v)
		}
		return a, nil
	case netip.Addr:
		if !v.Is4() {
			return netip.Addr{}, fmt.Errorf("not an IPv4 address: %v", v)
		}
		return v, nil
	case net.IP:
		ipv4Bytes := v.To4()
		if ipv4Bytes == nil {
			return netip.Addr{}, fmt.Errorf("not an IPv4 address: %v", v)
		}
		addr, ok := netip.AddrFromSlice(ipv4Bytes)
		if !ok {
			return netip.Addr{}, fmt.Errorf("not an IPv4 address: %v", v)
		}
		return addr, nil
	}
	panic(fmt.Sprintf("mustbe.IPv4: unhandled type: %T", a))
}

func IPv6(a any) (netip.Addr, error) {
	switch v := a.(type) {
	case string:
		a, err := netip.ParseAddr(v)
		if err != nil || !a.Is6() {
			return netip.Addr{}, fmt.Errorf("not a valid IPv6 address: %v", v)
		}
		return a, nil
	case netip.Addr:
		if !v.Is6() {
			return netip.Addr{}, fmt.Errorf("not an IPv6 address: %v", v)
		}
		return v, nil
	case net.IP:
		ipv6Bytes := v.To16()
		if ipv6Bytes == nil {
			return netip.Addr{}, fmt.Errorf("not a valid IPv6 address: %v", v)
		}
		addr, ok := netip.AddrFromSlice(ipv6Bytes)
		if !ok {
			return netip.Addr{}, fmt.Errorf("not a valid IPv6 address: %v", v)
		}
		return addr, nil

	}
	panic(fmt.Sprintf("mustbe.IPv6: unhandled type: %T", a))
}
