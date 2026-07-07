package models

import (
	dnsv2 "codeberg.org/miekg/dns"
)

// SetTargetSMIMEA sets the SMIMEA fields.
// Deprecated. Use models.NewRecordConfig() instead.
func (rc *RecordConfig) SetTargetSMIMEA(usage, selector, matchingtype uint8, target string) error {
	return legacySetTargetArgs(rc, dnsv2.TypeSMIMEA, usage, selector, matchingtype, target)
}

// // SetTargetSMIMEAStrings is like SetTargetSMIMEA but accepts strings.
// // Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) SetTargetSMIMEAStrings(usage, selector, matchingtype, target string) (err error) {
// 	return legacySetTargetArgs(rc, dnsv2.TypeSMIMEA, usage, selector, matchingtype, target)
// }

// // SetTargetSMIMEAString is like SetTargetSMIMEA but accepts one big string.
// // Deprecated. Use models.NewRecordConfigParse() instead.
// func (rc *RecordConfig) SetTargetSMIMEAString(s string) error {
// 	return legacySetTargetParse(rc, dnsv2.TypeSMIMEA, s)
// }
