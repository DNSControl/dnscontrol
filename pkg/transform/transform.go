package transform

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

// IPConversion describes an IP conversion.
type IPConversion struct {
	Low, High netip.Addr
	NewBases  []netip.Addr
	NewIPs    []netip.Addr
}

func ipToUint(i netip.Addr) (uint32, error) {
	if !i.Is4() {
		return 0, fmt.Errorf("%s is not an ipv4 address", i.String())
	}
	parts := i.AsSlice()
	r := uint32(parts[0])<<24 | uint32(parts[1])<<16 | uint32(parts[2])<<8 | uint32(parts[3])
	return r, nil
}

// UintToIP convert a 32-bit into a netip.Addr.
func UintToIP(u uint32) netip.Addr {
	return netip.AddrFrom4([4]byte{
		byte((u >> 24) & 255),
		byte((u >> 16) & 255),
		byte((u >> 8) & 255),
		byte((u) & 255),
	})
}

// DecodeTransformTable turns a string-encoded table into a list of conversions.
func DecodeTransformTable(transforms string) ([]IPConversion, error) {
	result := []IPConversion{}
	rows := strings.Split(transforms, ";")
	for ri, row := range rows {
		items := strings.Split(row, "~")
		if len(items) != 4 {
			return nil, fmt.Errorf("transform_table rows should have 4 elements. (%v) found in row (%v) of %#v", len(items), ri, transforms)
		}
		for i, item := range items {
			items[i] = strings.TrimSpace(item)
		}

		var err error
		var tLow, tHigh netip.Addr
		tLow, err = netip.ParseAddr(items[0])
		if err != nil {
			return nil, err
		}
		tHigh, err = netip.ParseAddr(items[1])
		if err != nil {
			return nil, err
		}

		con := IPConversion{
			Low:  tLow,
			High: tHigh,
		}
		parseList := func(s string) ([]netip.Addr, error) {
			ips := []netip.Addr{}
			for ip := range strings.SplitSeq(s, ",") {
				if ip == "" {
					continue
				}
				addr, err := netip.ParseAddr(ip)
				if err != nil {
					return nil, err
				}
				ips = append(ips, addr)
			}
			return ips, nil
		}
		//var err error
		if con.NewBases, err = parseList(items[2]); err != nil {
			return nil, err
		}
		if con.NewIPs, err = parseList(items[3]); err != nil {
			return nil, err
		}

		low, _ := ipToUint(con.Low)
		high, _ := ipToUint(con.High)
		if low > high {
			return nil, fmt.Errorf("transform_table Low should be less than High. row (%v) %v>%v (%v)", ri, con.Low, con.High, transforms)
		}
		if len(con.NewBases) > 0 && len(con.NewIPs) > 0 {
			return nil, errors.New("transform_table_rows should only specify one of NewBases or NewIPs, Not both")
		}
		result = append(result, con)
	}

	return result, nil
}

// IP transforms a single ip address. If the transform results in multiple new targets, an error will be returned.
func IP(address netip.Addr, transforms []IPConversion) (netip.Addr, error) {
	ips, err := IPToList(address, transforms)
	if err != nil {
		return netip.Addr{}, err
	}
	if len(ips) != 1 {
		return netip.Addr{}, fmt.Errorf("exactly one IP expected. Got: %s", ips)
	}
	return ips[0], err
}

// IPToList manipulates an net.IP based on a list of IPConversions. It can potentially expand one ip address into multiple addresses.
func IPToList(address netip.Addr, transforms []IPConversion) ([]netip.Addr, error) {
	thisIP, err := ipToUint(address)
	if err != nil {
		return nil, err
	}
	for _, conv := range transforms {
		minIP, err := ipToUint(conv.Low)
		if err != nil {
			return nil, err
		}
		maxIP, err := ipToUint(conv.High)
		if err != nil {
			return nil, err
		}
		if (thisIP >= minIP) && (thisIP <= maxIP) {
			if len(conv.NewIPs) > 0 {
				return conv.NewIPs, nil
			}
			list := []netip.Addr{}
			for _, nb := range conv.NewBases {
				newbase, err := ipToUint(nb)
				if err != nil {
					return nil, err
				}
				list = append(list, UintToIP(newbase+(thisIP-minIP)))
			}
			return list, nil
		}
	}
	return []netip.Addr{address}, nil
}
