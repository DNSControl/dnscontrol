package models

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
)

// SetTargetSRV sets the SRV fields.
// Deprecated. Use models.NewRecordConfig() instead.
func (rc *RecordConfig) SetTargetSRV(priority, weight, port uint16, target string) error {
	return legacySetTargetArgs(rc, dnsv2.TypeSRV, priority, weight, port, target)
}

// // setTargetSRVIntAndStrings is like SetTargetSRV but accepts priority as an int, the other parameters as strings.
// // Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) setTargetSRVIntAndStrings(priority uint16, weight, port, target string) (err error) {
// 	return legacySetTargetArgs(rc, dnsv2.TypeSRV, priority, weight, port, target)
// }

// // SetTargetSRVStrings is like SetTargetSRV but accepts all parameters as strings.
// // Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) SetTargetSRVStrings(priority, weight, port, target string) (err error) {
// 	return legacySetTargetArgs(rc, dnsv2.TypeSRV, priority, weight, port, target)
// }

// SetTargetSRVPriorityString is like SetTargetSRV but accepts priority as an
// uint16 and the rest of the values joined in a string that needs to be parsed.
// This is a helper function that comes in handy when a provider re-uses the MX preference
// field as the SRV priority.
// Deprecated. Use models.NewRecordConfigParse() instead.
func (rc *RecordConfig) SetTargetSRVPriorityString(priority uint16, s string) error {
	part := strings.Fields(s)
	switch len(part) {
	case 3:
		return legacySetTargetArgs(rc, dnsv2.TypeSRV, priority, part[0], part[1], part[2])
	case 2:
		return legacySetTargetArgs(rc, dnsv2.TypeSRV, priority, part[0], part[1], ".")
	default:
		return fmt.Errorf("SRV value does not contain 3 fields: (%#v)", s)
	}
}

// SetTargetSRVString is like SetTargetSRV but accepts one big string to be parsed.
// Deprecated. Use models.NewRecordConfigParse() instead.
func (rc *RecordConfig) SetTargetSRVString(s string) error {
	return legacySetTargetParse(rc, dnsv2.TypeSRV, s)
}
