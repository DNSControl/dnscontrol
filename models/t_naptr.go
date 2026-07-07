package models

import (
	dnsv2 "codeberg.org/miekg/dns"
)

// SetTargetNAPTR sets the NAPTR fields.
// Deprecated. Use models.NewRecordConfig() instead.
func (rc *RecordConfig) SetTargetNAPTR(order uint16, preference uint16, flags string, service string, regexp string, target string) error {
	return legacySetTargetArgs(rc, dnsv2.TypeNAPTR, order, preference, flags, service, regexp, target)
}

// // SetTargetNAPTRStrings is like SetTargetNAPTR but accepts strings.
// Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) SetTargetNAPTRStrings(order, preference, flags string, service string, regexp string, target string) error {
// 	return legacySetTargetArgs(rc, dnsv2.TypeNAPTR, order, preference, flags, service, regexp, target)
// }

// SetTargetNAPTRString is like SetTargetNAPTR but accepts one big string.
// Deprecated. Use models.NewRecordConfigParse() instead.
func (rc *RecordConfig) SetTargetNAPTRString(s string) error {
	return legacySetTargetParse(rc, dnsv2.TypeNAPTR, s)
}
