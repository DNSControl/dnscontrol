package models

import (
	dnsv2 "codeberg.org/miekg/dns"
)

// SetTargetCAA sets the CAA fields.
// Deprecated. Use models.NewRecordConfig() instead.
func (rc *RecordConfig) SetTargetCAA(flag uint8, tag string, target string) error {
	return legacySetTargetArgs(rc, dnsv2.TypeCAA, flag, tag, target)
}

// SetTargetCAAStrings is like SetTargetCAA but accepts strings.
// Deprecated. Use models.NewRecordConfig() instead.
func (rc *RecordConfig) SetTargetCAAStrings(flag, tag, target string) error {
	return legacySetTargetArgs(rc, dnsv2.TypeCAA, flag, tag, target)
}

// SetTargetCAAString is like SetTargetCAA but accepts one big string.
// Ex: `0 issue "letsencrypt.org"`.
// Deprecated. Use models.NewRecordConfigParse() instead.
func (rc *RecordConfig) SetTargetCAAString(s string) error {
	return legacySetTargetParse(rc, dnsv2.TypeCAA, s)
}
