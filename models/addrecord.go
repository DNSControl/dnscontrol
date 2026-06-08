package models

func (dc *DomainConfig) AddRecordConfig(rc *RecordConfig) {
	dc.Records = append(dc.Records, rc)
}

// func (dc *DomainConfig) AddRecord(label string, ttl uint32, rtyp uint16, args ...any) {
// 	rc := MakeRC(dc.Name, label, ttl, rtyp, args...)
// 	dc.Records = append(dc.Records, rc)
// }

// func MakeRC(origin string, label string, ttl uint32, rtyp uint16, args ...any) *RecordConfig {
// 	if ttl == 0 {
// 		ttl = DefaultTTL
// 	}
// 	rc := &RecordConfig{
// 		Type:    dnsutilv2.TypeToString(rtyp),
// 		TypeNum: rtyp,
// 		TTL:     ttl,
// 		Name:    label,
// 		RDATA:   makeRDATA(origin, rtyp, args...),
// 	}
// 	return rc
// }

// func makeRDATA(origin string, rtyp uint16, args ...any) dnsv2.RDATA {
// 	if ok, needed := correctNumberOfArgs(rtyp, len(args)); !ok {
// 		panic(fmt.Sprintf("makeRDATA: wrong number of args for type %s: expected %d, got %d (%+v)", dnsutilv2.TypeToString(rtyp), needed, len(args), args))
// 	}
// 	switch rtyp {
// 	case dnsv2.TypeA:
// 		return dnsrdatav2.A{Addr: mustbe.IPv4(args[0])}
// 	case dnsv2.TypeAAAA:
// 		return dnsrdatav2.AAAA{Addr: mustbe.IPv6(args[0])}
// 	case dnsv2.TypeCNAME:
// 		return dnsrdatav2.CNAME{Target: mustbe.TargetHost(origin, args[0])}
// 	case dnsv2.TypeTXT:
// 		return dnsrdatav2.TXT{Txt: mustbe.Txts(args)}

// 	case privatetypes.TypePORKBUNURLFWD:
// 		return privatetypesrdata.PORKBUNURLFWD{}
// 	}
// 	panic("makeRDATA: unhandled type " + dnsutilv2.TypeToString(rtyp))

// 	/*

// 	 */
// }

// func correctNumberOfArgs(rtyp uint16, n int) (bool, int) {
// 	var argsNeeded = map[uint16]int{
// 		// -1 == 1 or more
// 		// -2 == 0 or more
// 		// 0, 1, 2, ... == exact number of args
// 		dnsv2.TypeA:                    1,
// 		dnsv2.TypeAAAA:                 1,
// 		dnsv2.TypeCNAME:                1,
// 		dnsv2.TypeTXT:                  -2,
// 		privatetypes.TypePORKBUNURLFWD: 0,
// 	}

// 	needed, ok := argsNeeded[rtyp]
// 	if !ok {
// 		panic(fmt.Sprintf("correctNumberOfArgs: unhandled type %s", dnsutilv2.TypeToString(rtyp)))
// 	}
// 	if needed == -1 && n >= 1 {
// 		return true, n
// 	}
// 	if needed == -2 && n >= 0 {
// 		return true, n
// 	}
// 	return (needed == n), needed
// }

// // // NormalizeShort adds origin to s if s is not already a FQDN.
// // // Note that the result may not be a FQDN.  If origin does not end
// // // with a ".", the result won't either.
// // // This implements the zonefile convention (specified in RFC 1035,
// // // Section "5.1. Format") that "@" represents the
// // // apex (bare) domain. i.e. AddOrigin("@", "foo.com.") returns "foo.com.".
// // func NormalizeShort(s, origin string) string {
// // 	// ("foo.", "origin.") -> "foo." (already a FQDN)
// // 	// ("foo", "origin.") -> "foo.origin."
// // 	// ("foo", "origin") -> "foo.origin"
// // 	// ("foo", ".") -> "foo." (Same as dns.Fqdn())
// // 	// ("foo.", ".") -> "foo." (Same as dns.Fqdn())
// // 	// ("@", "origin.") -> "origin." (@ represents the apex (bare) domain)
// // 	// ("", "origin.") -> "origin." (not obvious)
// // 	// ("foo", "") -> "foo" (not obvious)

// // 	if dnsutilv2.IsFqdn(s) {
// // 		return s // s is already a FQDN, no need to mess with it.
// // 	}
// // 	if origin == "" {
// // 		return s // Nothing to append.
// // 	}
// // 	if s == "@" || s == "" {
// // 		return origin // Expand apex.
// // 	}
// // 	if origin == "." {
// // 		return dnsutilv2.Fqdn(s)
// // 	}

// // 	return s + "." + origin // The simple case.
// // }
