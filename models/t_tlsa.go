package models

import (
	"fmt"
	"strconv"
	"strings"
)

// SetTargetTLSA sets the TLSA fields.
func (rc *RecordConfig) SetTargetTLSA(usage, selector, matchingtype uint8, target string) error {
	rc.TlsaUsage = usage
	rc.TlsaSelector = selector
	rc.TlsaMatchingType = matchingtype
	if err := rc.SetTarget(target); err != nil {
		return err
	}
	if rc.Type == "" {
		rc.Type = "TLSA"
	}
	if rc.Type != "TLSA" {
		panic("assertion failed: SetTargetTLSA called when .Type is not TLSA")
	}
	return nil
}

// SetTargetTLSAStrings is like SetTargetTLSA but accepts strings.
func (rc *RecordConfig) SetTargetTLSAStrings(usage, selector, matchingtype, target string) (err error) {
	var i64usage, i64selector, i64matchingtype uint64
	if i64usage, err = strconv.ParseUint(usage, 10, 8); err == nil {
		if i64selector, err = strconv.ParseUint(selector, 10, 8); err == nil {
			if i64matchingtype, err = strconv.ParseUint(matchingtype, 10, 8); err == nil {
				return rc.SetTargetTLSA(uint8(i64usage), uint8(i64selector), uint8(i64matchingtype), target)
			}
		}
	}
	return fmt.Errorf("TLSA has value that won't fit in field: %w", err)
}

// SetTargetTLSAString is like SetTargetTLSA but accepts one big string.
func (rc *RecordConfig) SetTargetTLSAString(s string) error {
	part := strings.Fields(s)
	if len(part) != 4 {
		return fmt.Errorf("TLSA value does not contain 4 fields: (%#v)", s)
	}
	return rc.SetTargetTLSAStrings(part[0], part[1], part[2], part[3])
}
