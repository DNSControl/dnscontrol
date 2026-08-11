package models

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// SetTargetDS sets the DS fields.
func (rc *RecordConfig) SetTargetDS(keytag uint16, algorithm, digesttype uint8, digest string) error {
	rc.DsKeyTag = keytag
	rc.DsAlgorithm = algorithm
	rc.DsDigestType = digesttype
	rc.DsDigest = digest

	if rc.Type == "" {
		rc.Type = "DS"
	}
	if rc.Type != "DS" {
		panic("assertion failed: SetTargetDS called when .Type is not DS")
	}

	return nil
}

// SetTargetDSStrings is like SetTargetDS but accepts strings.
func (rc *RecordConfig) SetTargetDSStrings(keytag, algorithm, digesttype, digest string) error {
	u16keytag, err := strconv.ParseUint(keytag, 10, 16)
	if err != nil {
		return errors.Wrap(err, "DS KeyTag can't fit in 16 bits")
	}
	u8algorithm, err := strconv.ParseUint(algorithm, 10, 8)
	if err != nil {
		return errors.Wrap(err, "DS Algorithm can't fit in 8 bits")
	}
	u8digesttype, err := strconv.ParseUint(digesttype, 10, 8)
	if err != nil {
		return errors.Wrap(err, "DS DigestType can't fit in 8 bits")
	}

	return rc.SetTargetDS(uint16(u16keytag), uint8(u8algorithm), uint8(u8digesttype), digest)
}

// SetTargetDSString is like SetTargetDS but accepts one big string.
func (rc *RecordConfig) SetTargetDSString(s string) error {
	part := strings.Fields(s)
	if len(part) != 4 {
		return errors.Errorf("DS value does not contain 4 fields: (%#v)", s)
	}
	return rc.SetTargetDSStrings(part[0], part[1], part[2], part[3])
}
