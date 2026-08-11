package models

import (
	"fmt"
	"strconv"
	"strings"
)

// SetTargetNAPTR sets the NAPTR fields.
func (rc *RecordConfig) SetTargetNAPTR(order uint16, preference uint16, flags string, service string, regexp string, target string) error {
	if target == "" {
		target = "."
	}
	rc.NaptrOrder = order
	rc.NaptrPreference = preference
	rc.NaptrFlags = flags
	rc.NaptrService = service
	rc.NaptrRegexp = regexp
	if err := rc.SetTarget(target); err != nil {
		return err
	}

	if rc.Type == "" {
		rc.Type = "NAPTR"
	}
	if rc.Type != "NAPTR" {
		panic("assertion failed: SetTargetNAPTR called when .Type is not NAPTR")
	}

	return nil
}

// SetTargetNAPTRStrings is like SetTargetNAPTR but accepts strings.
func (rc *RecordConfig) SetTargetNAPTRStrings(order, preference, flags string, service string, regexp string, target string) error {
	i64order, err := strconv.ParseUint(order, 10, 16)
	if err != nil {
		return fmt.Errorf("NAPTR order does not fit in 16 bits: %w", err)
	}
	i64preference, err := strconv.ParseUint(preference, 10, 16)
	if err != nil {
		return fmt.Errorf("NAPTR preference does not fit in 16 bits: %w", err)
	}
	return rc.SetTargetNAPTR(uint16(i64order), uint16(i64preference), flags, service, regexp, target)
}

// SetTargetNAPTRString is like SetTargetNAPTR but accepts one big string.
func (rc *RecordConfig) SetTargetNAPTRString(s string) error {
	part := strings.Fields(s)
	if len(part) != 6 {
		return fmt.Errorf("NAPTR value does not contain 6 fields: (%#v)", s)
	}
	return rc.SetTargetNAPTRStrings(part[0], part[1], StripQuotes(part[2]), StripQuotes(part[3]), StripQuotes(part[4]), StripQuotes(part[5]))
}
