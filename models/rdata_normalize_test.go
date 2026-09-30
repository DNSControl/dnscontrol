package models

import (
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
	"github.com/google/go-cmp/cmp"
)

func TestRDATANormalization(t *testing.T) {
	for _, tc := range []struct {
		name, rtype string
		input, want dnsv2.RDATA
	}{
		{
			"hostname case", "MX",
			dnsrdatav2.MX{Preference: 10, Mx: "MAIL.xn--bcher-kva.Example."},
			dnsrdatav2.MX{Preference: 10, Mx: "mail.xn--bcher-kva.example."},
		},
		{
			"hostname without rewriting mailbox", "SOA",
			dnsrdatav2.SOA{Ns: "NS.Example.", Mbox: "HostMaster.Example.", Serial: 42},
			dnsrdatav2.SOA{Ns: "ns.example.", Mbox: "HostMaster.Example.", Serial: 42},
		},
		{
			"private hostname without rewriting other fields", "R53_ALIAS",
			privatetypesrdata.R53ALIAS{AliasType: "A", Target: "HOST.Example.", ZoneID: "ABC123"},
			privatetypesrdata.R53ALIAS{AliasType: "A", Target: "host.example.", ZoneID: "ABC123"},
		},
		{
			"hostname slice", "HIP",
			dnsrdatav2.HIP{RendezvousServers: []string{"unchanged.example.", "HOST.Example.", "xn--bcher-kva.Example."}},
			dnsrdatav2.HIP{RendezvousServers: []string{"unchanged.example.", "host.example.", "xn--bcher-kva.example."}},
		},
		{
			"nil hostname slice", "HIP",
			dnsrdatav2.HIP{}, dnsrdatav2.HIP{},
		},
		{
			"hex digest", "DS",
			dnsrdatav2.DS{KeyTag: 42, Digest: "ABcd0123"},
			dnsrdatav2.DS{KeyTag: 42, Digest: "abcd0123"},
		},
		{
			"hex certificate", "TLSA",
			dnsrdatav2.TLSA{Usage: 3, Selector: 1, MatchingType: 1, Certificate: "ABcd0123"},
			dnsrdatav2.TLSA{Usage: 3, Selector: 1, MatchingType: 1, Certificate: "abcd0123"},
		},
		{
			"additional hex certificate", "SMIMEA",
			dnsrdatav2.SMIMEA{Usage: 3, Certificate: "ABcd0123"},
			dnsrdatav2.SMIMEA{Usage: 3, Certificate: "abcd0123"},
		},
		{
			"additional hex digest", "ZONEMD",
			dnsrdatav2.ZONEMD{Digest: "ABcd0123"},
			dnsrdatav2.ZONEMD{Digest: "abcd0123"},
		},
		{
			"root target", "HTTPS",
			dnsrdatav2.SVCB{Priority: 1, Target: "."},
			dnsrdatav2.SVCB{Priority: 1, Target: "."},
		},
		{
			"empty SRV target", "SRV",
			dnsrdatav2.SRV{Target: ""}, dnsrdatav2.SRV{Target: ""},
		},
		{
			"mailbox and opaque name", "RP",
			dnsrdatav2.RP{Mbox: "Admin.Example.", Txt: "Info.Example."},
			dnsrdatav2.RP{Mbox: "Admin.Example.", Txt: "Info.Example."},
		},
		{
			"DNSSEC opaque name", "RRSIG",
			dnsrdatav2.RRSIG{SignerName: "Signer.Example."},
			dnsrdatav2.RRSIG{SignerName: "Signer.Example."},
		},
		{
			"private URL", "BUNNY_DNS_RDR",
			privatetypesrdata.BUNNYDNSRDR{Target: "https://Example.com/CaseSensitive"},
			privatetypesrdata.BUNNYDNSRDR{Target: "https://Example.com/CaseSensitive"},
		},
		{
			"obsolete type", "MB",
			dnsrdatav2.MB{Mb: "HOST.Example."}, dnsrdatav2.MB{Mb: "HOST.Example."},
		},
		{
			"TXT segmentation and case", "TXT",
			dnsrdatav2.TXT{Txt: []string{strings.Repeat("A", 254), "BC"}},
			dnsrdatav2.TXT{Txt: []string{strings.Repeat("A", 254) + "B", "C"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.input.String()
			rc, err := MustNewDomainConfig("example.com").NewRecordConfigForRRv2toRC("@", 300, dnsv2.StringToType[tc.rtype], tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, rc.GetRDATA()); diff != "" {
				t.Errorf("normalized RDATA mismatch (-want +got):\n%s", diff)
			}
			if tc.input.String() != before {
				t.Error("normalization mutated the caller's RDATA")
			}
			comp := rc.ComparableV3
			rc.SetRDATA(rc.GetRDATA())
			if diff := cmp.Diff(tc.want, rc.GetRDATA()); diff != "" || comp != rc.ComparableV3 {
				t.Errorf("normalization is not idempotent: %s", diff)
			}
		})
	}
}

func TestRDATANormalizationAcrossConstructors(t *testing.T) {
	dc := MustNewDomainConfig("example.com")
	for _, tc := range []struct {
		rtype string
		args  []any
		text  string
		rdata dnsv2.RDATA
	}{
		{"MX", []any{10, "MAIL.Example.NET."}, "10 MAIL.Example.NET.", dnsrdatav2.MX{Preference: 10, Mx: "MAIL.Example.NET."}},
		// Makers convert Unicode input before normalization; parsed/imported RDATA is ASCII.
		{"MX", []any{10, "MAIL.bücher.Example."}, "10 MAIL.xn--bcher-kva.Example.", dnsrdatav2.MX{Preference: 10, Mx: "MAIL.xn--bcher-kva.Example."}},
		{"ALIAS", []any{"HOST.Example.NET."}, "HOST.Example.NET.", privatetypesrdata.ALIAS{Target: "HOST.Example.NET."}},
		{"TLSA", []any{3, 1, 1, "ABCD0123"}, "3 1 1 ABCD0123", dnsrdatav2.TLSA{Usage: 3, Selector: 1, MatchingType: 1, Certificate: "ABCD0123"}},
	} {
		t.Run(tc.rtype, func(t *testing.T) {
			made := dc.MustNewRecordConfig("www", 300, tc.rtype, tc.args...)
			parsed := dc.MustNewRecordConfigParse("www", 300, tc.rtype, tc.text)
			imported, err := dc.NewRecordConfigForRRv2toRC("www", 300, dnsv2.StringToType[tc.rtype], tc.rdata)
			if err != nil {
				t.Fatal(err)
			}
			for name, rc := range map[string]*RecordConfig{"parsed": parsed, "imported": imported} {
				if diff := cmp.Diff(made.GetRDATA(), rc.GetRDATA()); diff != "" {
					t.Errorf("%s differs from maker (-want +got):\n%s", name, diff)
				}
				if rc.ComparableV3 != made.ComparableV3 {
					t.Errorf("%s has a different comparison value", name)
				}
			}
		})
	}
}

func TestRDATANormalizationUnchangedAllocations(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input dnsv2.RDATA
	}{
		{"hostname", dnsrdatav2.MX{Mx: "mail.example.com."}},
		{"punycode", dnsrdatav2.MX{Mx: "mail.xn--bcher-kva.example."}},
		{"hex", dnsrdatav2.TLSA{Certificate: strings.Repeat("0123456789abcdef", 4)}},
		{"txt", dnsrdatav2.TXT{Txt: []string{"already normalized"}}},
		{"segmented txt", dnsrdatav2.TXT{Txt: []string{strings.Repeat("A", 255), "B"}}},
		{"empty txt", dnsrdatav2.TXT{Txt: []string{""}}},
		{"hostname slice", dnsrdatav2.HIP{RendezvousServers: []string{"one.example.", "two.example."}}},
		{"nil slice", dnsrdatav2.HIP{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			allocs := testing.AllocsPerRun(100, func() {
				_ = normalizeRDATA(tc.input)
			})
			if allocs != 0 {
				t.Errorf("unchanged RDATA allocated %g times, want 0", allocs)
			}
		})
	}
}
