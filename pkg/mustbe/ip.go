package mustbe

import (
	"fmt"
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
	}
	panic(fmt.Sprintf("mustbe.IPv6: unhandled type: %T", a))
}
