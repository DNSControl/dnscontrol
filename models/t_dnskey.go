package models

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// SetTargetDNSKEY sets the DNSKEY fields.
func (rc *RecordConfig) SetTargetDNSKEY(flags uint16, protocol, algorithm uint8, publicKey string) error {
	rc.DnskeyFlags = flags
	rc.DnskeyProtocol = protocol
	rc.DnskeyAlgorithm = algorithm
	rc.DnskeyPublicKey = publicKey

	if rc.Type == "" {
		rc.Type = "DNSKEY"
	}
	if rc.Type != "DNSKEY" {
		panic("assertion failed: SetTargetDNSKEY called when .Type is not DNSKEY")
	}

	return nil
}

// SetTargetDNSKEYStrings is like SetTargetDNSKEY but accepts strings.
func (rc *RecordConfig) SetTargetDNSKEYStrings(flags, protocol, algorithm, publicKey string) error {
	u16flags, err := strconv.ParseUint(flags, 10, 16)
	if err != nil {
		return errors.Wrap(err, "DNSKEY Flags can't fit in 16 bits")
	}
	u8protocol, err := strconv.ParseUint(protocol, 10, 8)
	if err != nil {
		return errors.Wrap(err, "DNSKEY Protocol can't fit in 8 bits")
	}
	u8algorithm, err := strconv.ParseUint(algorithm, 10, 8)
	if err != nil {
		return errors.Wrap(err, "DNSKEY Algorithm can't fit in 8 bits")
	}

	return rc.SetTargetDNSKEY(uint16(u16flags), uint8(u8protocol), uint8(u8algorithm), publicKey)
}

// SetTargetDNSKEYString is like SetTargetDNSKEY but accepts one big string.
func (rc *RecordConfig) SetTargetDNSKEYString(s string) error {
	part := strings.Fields(s)
	if len(part) != 4 {
		return errors.Errorf("DNSKEY value does not contain 4 fieldnskey: (%#v)", s)
	}
	return rc.SetTargetDNSKEYStrings(part[0], part[1], part[2], part[3])
}
