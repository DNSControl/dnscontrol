package models

import (
	"fmt"
	"strconv"
	"strings"
)

// SetTargetSRV sets the SRV fields.
func (rc *RecordConfig) SetTargetSRV(priority, weight, port uint16, target string) error {
	rc.SrvPriority = priority
	rc.SrvWeight = weight
	rc.SrvPort = port
	if err := rc.SetTarget(target); err != nil {
		return err
	}
	if rc.Type == "" {
		rc.Type = "SRV"
	}
	if rc.Type != "SRV" {
		panic("assertion failed: SetTargetSRV called when .Type is not SRV")
	}
	return nil
}

// setTargetSRVIntAndStrings is like SetTargetSRV but accepts priority as an int, the other parameters as strings.
func (rc *RecordConfig) setTargetSRVIntAndStrings(priority uint16, weight, port, target string) (err error) {
	var i64weight, i64port uint64
	if i64weight, err = strconv.ParseUint(weight, 10, 16); err == nil {
		if i64port, err = strconv.ParseUint(port, 10, 16); err == nil {
			return rc.SetTargetSRV(priority, uint16(i64weight), uint16(i64port), target)
		}
	}
	return fmt.Errorf("SRV value too big for uint16: %w", err)
}

// SetTargetSRVStrings is like SetTargetSRV but accepts all parameters as strings.
func (rc *RecordConfig) SetTargetSRVStrings(priority, weight, port, target string) (err error) {
	var i64priority uint64
	if i64priority, err = strconv.ParseUint(priority, 10, 16); err == nil {
		return rc.setTargetSRVIntAndStrings(uint16(i64priority), weight, port, target)
	}
	return fmt.Errorf("SRV value too big for uint16: %w", err)
}

// SetTargetSRVPriorityString is like SetTargetSRV but accepts priority as an
// uint16 and the rest of the values joined in a string that needs to be parsed.
// This is a helper function that comes in handy when a provider re-uses the MX preference
// field as the SRV priority.
func (rc *RecordConfig) SetTargetSRVPriorityString(priority uint16, s string) error {
	part := strings.Fields(s)
	switch len(part) {
	case 3:
		return rc.setTargetSRVIntAndStrings(priority, part[0], part[1], part[2])
	case 2:
		return rc.setTargetSRVIntAndStrings(priority, part[0], part[1], ".")
	default:
		return fmt.Errorf("SRV value does not contain 3 fields: (%#v)", s)
	}
}

// SetTargetSRVString is like SetTargetSRV but accepts one big string to be parsed.
func (rc *RecordConfig) SetTargetSRVString(s string) error {
	part := strings.Fields(s)
	if len(part) != 4 {
		return fmt.Errorf("SRV value does not contain 4 fields: (%#v)", s)
	}
	return rc.SetTargetSRVStrings(part[0], part[1], part[2], part[3])
}
