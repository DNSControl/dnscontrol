package models

import dnsv2 "codeberg.org/miekg/dns"

/*

Providers are not expected to support this record.

Most providers do not support SOA records. They generate them
dynamically behind the scenes.  Providers like BIND (which is
software, not SaaS), must handle SOA records and emulate the dynamic
work that providers do.

*/

// SetTargetSOA sets the SOA fields.
// Deprecated. Use models.NewRecordConfig() instead.
func (rc *RecordConfig) SetTargetSOA(ns, mbox string, serial, refresh, retry, expire, minttl uint32) error {
	return legacySetTargetArgs(rc, dnsv2.TypeSOA, ns, mbox, serial, refresh, retry, expire, minttl)
}

// // SetTargetSOAStrings is like SetTargetSOA but accepts strings.
// // Deprecated. Use models.NewRecordConfig() instead.
// func (rc *RecordConfig) SetTargetSOAStrings(ns, mbox, serial, refresh, retry, expire, minttl string) error {
// 	return legacySetTargetArgs(rc, dnsv2.TypeSOA, ns, mbox, serial, refresh, retry, expire, minttl)
// }

// // SetTargetSOAString is like SetTargetSOA but accepts one big string.
// // Deprecated. Use models.NewRecordConfigParse() instead.
// func (rc *RecordConfig) SetTargetSOAString(s string) error {
// 	return legacySetTargetParse(rc, dnsv2.TypeSOA, s)
// }
