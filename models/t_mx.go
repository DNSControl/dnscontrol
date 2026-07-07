package models

import (
	dnsv2 "codeberg.org/miekg/dns"
)

// SetTargetMX sets the MX fields.
// Deprecated. Use models.NewRecordConfig() instead.
func (rc *RecordConfig) SetTargetMX(pref uint16, target string) error {
	return legacySetTargetArgs(rc, dnsv2.TypeMX, pref, target)
}

// // SetTargetMXStrings is like SetTargetMX but accepts strings.
// Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) SetTargetMXStrings(pref, target string) error {
// 	return legacySetTargetArgs(rc, dnsv2.TypeMX, pref, target)
// }

// SetTargetMXString is like SetTargetMX but accepts one big string.
// // Deprecated. Use models.NewRecordConfigParse() instead.
func (rc *RecordConfig) SetTargetMXString(s string) error {
	return legacySetTargetParse(rc, dnsv2.TypeMX, s)
}
