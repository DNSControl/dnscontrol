package models

import (
	dnsv2 "codeberg.org/miekg/dns"
)

// // SetTargetDS sets the DS fields.
// // Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) SetTargetDS(keytag uint16, algorithm, digesttype uint8, digest string) error {
// 	return legacySetTargetArgs(rc, dnsv2.TypeDS, keytag, algorithm, digesttype, digest)
// }

// // SetTargetDSStrings is like SetTargetDS but accepts strings.
// // Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) SetTargetDSStrings(keytag, algorithm, digesttype, digest string) error {
// 	return legacySetTargetArgs(rc, dnsv2.TypeDS, keytag, algorithm, digesttype, digest)
// }

// SetTargetDSString is like SetTargetDS but accepts one big string.
// Deprecated. Use models.NewRecordConfigParse() instead.
func (rc *RecordConfig) SetTargetDSString(s string) error {
	return legacySetTargetParse(rc, dnsv2.TypeDS, s)
}
