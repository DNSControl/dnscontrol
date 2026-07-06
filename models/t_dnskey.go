package models

import (
	dnsv2 "codeberg.org/miekg/dns"
)

// // SetTargetDNSKEY sets the DNSKEY fields.
// // Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) SetTargetDNSKEY(flags uint16, protocol, algorithm uint8, publicKey string) error {
// 	return legacySetTargetArgs(rc, dnsv2.TypeDNSKEY, flags, protocol, algorithm, publicKey)
// }

// // SetTargetDNSKEYStrings is like SetTargetDNSKEY but accepts strings.
// // Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) SetTargetDNSKEYStrings(flags, protocol, algorithm, publicKey string) error {
// 	return legacySetTargetArgs(rc, dnsv2.TypeDNSKEY, flags, protocol, algorithm, publicKey)
// }

// SetTargetDNSKEYString is like SetTargetDNSKEY but accepts one big string.
// Deprecated. Use models.NewRecordConfigParse() instead.
func (rc *RecordConfig) SetTargetDNSKEYString(s string) error {
	return legacySetTargetParse(rc, dnsv2.TypeDNSKEY, s)
}
