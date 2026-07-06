package mustbe

import (
	"fmt"
	"net"
	"net/netip"
)

func IPv4(a any) netip.Addr {
	switch v := a.(type) {
	case float64:
		i := int32(v)
		a := (i >> 24) % 256
		b := (i >> 16) % 256
		c := (i >> 8) % 256
		d := i % 256
		x := netip.AddrFrom4([4]byte{
			byte(a),
			byte(b),
			byte(c),
			byte(d),
		})
		return x
	case string:
		a, err := netip.ParseAddr(v)
		if err != nil || !a.Is4() {
			return netip.Addr{}
		}
		return a
	case netip.Addr:
		if !v.Is4() {
			return netip.Addr{}
		}
		return v
	case net.IP:
		ipv4Bytes := v.To4()
		if ipv4Bytes == nil {
			panic(fmt.Sprintf("not a valid IPv4 address: %v", v))
		}
		addr, ok := netip.AddrFromSlice(ipv4Bytes)
		if !ok {
			panic(fmt.Sprintf("failed to convert IPv4 address: %v", v))
		}
		return addr
	}
	panic(fmt.Sprintf("mustbe.IPv4: unhandled type: %T", a))
}

func IPv6(a any) netip.Addr {
	switch v := a.(type) {
	case string:
		a, err := netip.ParseAddr(v)
		if err != nil || !a.Is6() {
			return netip.Addr{}
		}
		return a
	case netip.Addr:
		if !v.Is6() {
			return netip.Addr{}
		}
		return v
	case net.IP:
		ipv6Bytes := v.To16()
		if ipv6Bytes == nil {
			panic(fmt.Sprintf("not a valid IPv6 address: %v", v))
		}
		addr, ok := netip.AddrFromSlice(ipv6Bytes)
		if !ok {
			panic(fmt.Sprintf("failed to convert IPv6 address: %v", v))
		}
		return addr

	}
	panic(fmt.Sprintf("mustbe.IPv6: unhandled type: %T", a))
}
