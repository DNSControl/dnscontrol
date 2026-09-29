package models

import (
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
)

func BenchmarkNormalizeRDATA(b *testing.B) {
	for _, tc := range []struct {
		name  string
		input dnsv2.RDATA
	}{
		{"hostname/unchanged", dnsrdatav2.MX{Preference: 10, Mx: "mail.example.com."}},
		{"hostname/uppercase", dnsrdatav2.MX{Preference: 10, Mx: "MAIL.Example.COM."}},
		{"hostname/punycode", dnsrdatav2.MX{Preference: 10, Mx: "mail.xn--bcher-kva.example."}},
		{"hex/unchanged", dnsrdatav2.TLSA{Certificate: strings.Repeat("0123456789abcdef", 4)}},
		{"hex/uppercase", dnsrdatav2.TLSA{Certificate: strings.Repeat("0123456789ABCDEF", 4)}},
		{"txt/unchanged", dnsrdatav2.TXT{Txt: []string{"already normalized"}}},
		{"txt/segmented", dnsrdatav2.TXT{Txt: []string{strings.Repeat("A", 255), "B"}}},
		{"txt/resegment", dnsrdatav2.TXT{Txt: []string{strings.Repeat("A", 254), "BC"}}},
		{"hip/unchanged", dnsrdatav2.HIP{RendezvousServers: []string{"one.example.", "two.example."}}},
		{"hip/changed", dnsrdatav2.HIP{RendezvousServers: []string{"one.example.", "TWO.Example.", "THREE.Example."}}},
	} {
		b.Run(tc.name, func(b *testing.B) {
			var rc RecordConfig
			b.ReportAllocs()
			for b.Loop() {
				// Reset every time so the changed cases keep measuring conversion.
				rc.rdata = tc.input
				rc.normalizeRDATA()
			}
		})
	}
}
