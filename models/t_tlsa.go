package models

import (
	dnsv2 "codeberg.org/miekg/dns"
)

// SetTargetTLSA sets the TLSA fields.
func (rc *RecordConfig) SetTargetTLSA(usage, selector, matchingtype uint8, target string) error {
	return legacySetTargetArgs(rc, dnsv2.TypeTLSA, usage, selector, matchingtype, target)
}

// // SetTargetTLSAStrings is like SetTargetTLSA but accepts strings.
// func (rc *RecordConfig) SetTargetTLSAStrings(usage, selector, matchingtype, target string) (err error) {
// 	return legacySetTargetArgs(rc, dnsv2.TypeTLSA, usage, selector, matchingtype, target)
// }

// SetTargetTLSAString is like SetTargetTLSA but accepts one big string.
func (rc *RecordConfig) SetTargetTLSAString(s string) error {
	return legacySetTargetParse(rc, dnsv2.TypeTLSA, s)
}
