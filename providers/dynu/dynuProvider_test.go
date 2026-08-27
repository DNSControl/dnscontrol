package dynu

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestToRc(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}

	priority := 10
	zero := 0
	weight := 2
	port := 5060
	order := 100
	preference := 10
	tests := []struct {
		name       string
		record     dynuRecord
		wantTarget string
	}{
		{"A", dynuRecord{NodeName: "www", RecordType: "A", IPv4Address: "192.0.2.1", TTL: 300}, "192.0.2.1"},
		{"MX", dynuRecord{NodeName: "@", RecordType: "MX", Host: "mail.example.net", Priority: &priority, TTL: 300}, "10 mail.example.net."},
		{"null MX", dynuRecord{NodeName: "@", RecordType: "MX", Host: "example.com", Priority: &zero, TTL: 300}, "0 ."},
		{"TXT", dynuRecord{NodeName: "@", RecordType: "TXT", TextData: "raw text", TTL: 300}, `"raw text"`},
		{"SRV", dynuRecord{NodeName: "_sip._tcp", RecordType: "SRV", Host: "sip.example.net", Priority: &priority, Weight: &weight, Port: &port, TTL: 300}, "10 2 5060 sip.example.net."},
		{"NAPTR", dynuRecord{NodeName: "@", RecordType: "NAPTR", Order: &order, Preference: &preference, NaptrFlags: "U", Services: "E2U+sip", RegExp: `!^.*$!sip:info@example.com!`, Replacement: "sip.example.net", TTL: 300}, `100 10 "U" "E2U+sip" "!^.*$!sip:info@example.com!" sip.example.net.`},
		{"NAPTR null replacement", dynuRecord{NodeName: "@", RecordType: "NAPTR", Order: &order, Preference: &preference, NaptrFlags: "S", Services: "SIP+D2U", Replacement: "", TTL: 300}, `100 10 "S" "SIP+D2U" "" .`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.record
			rc, err := toRc(&record, dc)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
			if rc.Original != &record {
				t.Error("original provider record was not retained")
			}
		})
	}
}

// TestToReqNAPTR verifies that toReq puts each NAPTR field in the matching Dynu
// API field. In particular, replacement must come from the record's replacement
// and not from its service.
func TestToReqNAPTR(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name            string
		replacement     string
		wantReplacement string
	}{
		{"fqdn replacement", "sip.example.net.", "sip.example.net"},
		{"null replacement", ".", "."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := dc.NewRecordConfig("@", 300, dnsv2.TypeNAPTR,
				100, 10, "U", "E2U+sip", `!^.*$!sip:info@example.com!`, tc.replacement) // ignore:legacyfield
			if err != nil {
				t.Fatal(err)
			}

			req := toReq(rc)
			if req.Replacement != tc.wantReplacement {
				t.Errorf("Replacement = %q, want %q", req.Replacement, tc.wantReplacement)
			}
			if req.Services != "E2U+sip" {
				t.Errorf("Services = %q, want %q", req.Services, "E2U+sip")
			}
			if req.NaptrFlags != "U" {
				t.Errorf("NaptrFlags = %q, want %q", req.NaptrFlags, "U")
			}
			if req.RegExp != `!^.*$!sip:info@example.com!` {
				t.Errorf("RegExp = %q, want %q", req.RegExp, `!^.*$!sip:info@example.com!`)
			}
			if intOrZero(req.Order) != 100 {
				t.Errorf("Order = %d, want 100", intOrZero(req.Order))
			}
			if intOrZero(req.Preference) != 10 {
				t.Errorf("Preference = %d, want 10", intOrZero(req.Preference))
			}
		})
	}
}

// TestToReqRP verifies that toReq strips the trailing dot from the RP mailbox
// and TXT domain name. The Dynu API validates both as hostnames and rejects a
// trailing dot with "Invalid content.". DNSControl fully qualifies relative
// names, so both fields always arrive here with a trailing dot.
func TestToReqRP(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name              string
		mbox, txt         string
		wantMbox, wantTxt string
	}{
		{"absolute names", "user.example.com.", "bar.com.", "user.example.com", "bar.com"},
		{"relative names get qualified", "user", "server", "user.example.com", "server.example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := dc.NewRecordConfig("foo", 300, dnsv2.TypeRP, tc.mbox, tc.txt)
			if err != nil {
				t.Fatal(err)
			}

			req := toReq(rc)
			if req.MailBox != tc.wantMbox {
				t.Errorf("MailBox = %q, want %q", req.MailBox, tc.wantMbox)
			}
			if req.TxtDomainName != tc.wantTxt {
				t.Errorf("TxtDomainName = %q, want %q", req.TxtDomainName, tc.wantTxt)
			}
		})
	}
}
