package transform

import (
	"encoding/base64"
	"encoding/hex"
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

var b64 = base64.StdEncoding.Strict()

// The target of an OPENPGPKEY record can be either hex or base64, so we need to
// be able to decode both formats.
//
// PGP keys are quite long and are largely random, so the odds of the base64
// encoding of a PGP key also being valid hex are very low. Therefore, we will
// assume that if a string is both valid hex and valid base64, then it is hex.
// The other cases where a string decodes as only hex, only base64, or neither
// are all unambiguous.
func decodeHexOrBase64(s string) ([]byte, error) {
	// A string with mixed casing *could* be hex, but it's more likely to be
	// base64. So we only try to decode as hex only if the string is all a
	// single case.
	var (
		hexErr  error
		hexData []byte
	)
	if s == strings.ToLower(s) || s == strings.ToUpper(s) {
		hexData, hexErr = hex.DecodeString(s)
	} else {
		hexData, hexErr = nil, fmt.Errorf(
			"hex string contains mixed case: %#v", s,
		)
	}

	// Also try to decode the string as base64, using the strictest possible
	// decoder (which is also used in the DNS "presentation" format and the "gpg
	// --armor" format).
	var (
		b64Err  error
		b64Data []byte
	)
	// Reject base64 strings that contain whitespace
	if strings.ContainsAny(s, " \t\r\n") {
		b64Data, b64Err = nil, fmt.Errorf(
			"base64 string contains whitespace: %#v", s,
		)
	} else {
		b64Data, b64Err = b64.DecodeString(s)
	}

	// Return the result.
	if hexErr != nil && b64Err != nil {
		// Both decodings failed, so there's nothing that we can do but return
		// an error.
		return nil, fmt.Errorf(
			"string is neither valid hex nor valid base64: %w; %w",
			hexErr, b64Err,
		)
	} else if hexErr == nil && b64Err != nil {
		// Only the hex decoding succeeded, therefore the input must only be
		// valid hex.
		return hexData, nil
	} else if hexErr != nil && b64Err == nil {
		// Only the base64 decoding succeeded, therefore the input must only be
		// valid base64.
		return b64Data, nil
	} else if hexErr == nil && b64Err == nil {
		// Both decodings succeeded. This is theoretically ambiguous, but it's
		// very unlikely that a valid base64 string would also be valid hex, so
		// we will assume that the input is hex.
		return hexData, nil
	} else {
		return nil, fmt.Errorf("unreachable")
	}
}

// OPENPGPKEY decoded an OPENPGP key.
func OPENPGPKEY(encodedKey string) (string, error) {
	// Decode the key, which can be either hex or base64.
	decodedKey, err := decodeHexOrBase64(encodedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode OPENPGPKEY: %w", err)
	}

	// Re-encode the key as base64, since the input may have been hex.
	encodedKey = b64.EncodeToString(decodedKey)
	return encodedKey, nil
}
