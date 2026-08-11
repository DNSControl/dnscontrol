package models

import (
	"fmt"
	"strconv"
	"strings"
)

// SetTargetSMIMEA sets the SMIMEA fields.
func (rc *RecordConfig) SetTargetSMIMEA(usage, selector, matchingtype uint8, target string) error {
	rc.SmimeaUsage = usage
	rc.SmimeaSelector = selector
	rc.SmimeaMatchingType = matchingtype
	if err := rc.SetTarget(target); err != nil {
		return err
	}
	if rc.Type == "" {
		rc.Type = "SMIMEA"
	}
	if rc.Type != "SMIMEA" {
		panic("assertion failed: SetTargetSMIMEA called when .Type is not SMIMEA")
	}
	return nil
}

// SetTargetSMIMEAStrings is like SetTargetSMIMEA but accepts strings.
func (rc *RecordConfig) SetTargetSMIMEAStrings(usage, selector, matchingtype, target string) (err error) {
	var i64usage, i64selector, i64matchingtype uint64
	if i64usage, err = strconv.ParseUint(usage, 10, 8); err == nil {
		if i64selector, err = strconv.ParseUint(selector, 10, 8); err == nil {
			if i64matchingtype, err = strconv.ParseUint(matchingtype, 10, 8); err == nil {
				return rc.SetTargetSMIMEA(uint8(i64usage), uint8(i64selector), uint8(i64matchingtype), target)
			}
		}
	}
	return fmt.Errorf("SMIMEA has value that won't fit in field: %w", err)
}

// SetTargetSMIMEAString is like SetTargetSMIMEA but accepts one big string.
func (rc *RecordConfig) SetTargetSMIMEAString(s string) error {
	part := strings.Fields(s)
	if len(part) != 4 {
		return fmt.Errorf("SMIMEA value does not contain 4 fields: (%#v)", s)
	}
	return rc.SetTargetSMIMEAStrings(part[0], part[1], part[2], part[3])
}
