package transip

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/transip/gotransip/v6/domain"
)

func TestNativeToRecordUsesV3RecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name string
		in   domain.DNSEntry
		want *models.RecordConfig
	}{
		{"A", domain.DNSEntry{Name: "www", Type: "A", Content: "192.0.2.1", Expire: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1")},
		// TransIP stores and returns TXT content unquoted, so it is parsed as raw data.
		{"TXT", domain.DNSEntry{Name: "www", Type: "TXT", Content: "hello world", Expire: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, "hello world")},
		{"MX", domain.DNSEntry{Name: "www", Type: "MX", Content: "10 mail.example.net.", Expire: 300}, dc.MustNewRecordConfig("www", 300, dnsv2.TypeMX, uint16(10), "mail.example.net.")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nativeToRecord(tt.in, dc)
			if err != nil {
				t.Fatal(err)
			}
			if got.NameFQDN != tt.want.NameFQDN || got.TTL != tt.want.TTL || got.TypeNum != tt.want.TypeNum || got.GetRDATA().String() != tt.want.GetRDATA().String() {
				t.Errorf("nativeToRecord() = %s %d IN %s %s, want %s %d IN %s %s", got.NameFQDN, got.TTL, got.Type, got.GetRDATA(), tt.want.NameFQDN, tt.want.TTL, tt.want.Type, tt.want.GetRDATA())
			}
		})
	}
}

// TestRecordToNativeTXTUnquoted verifies that TXT records are written to TransIP
// as the raw, unquoted value rather than the RFC presentation form. Writing the
// quoted form caused TransIP to serve literal double-quotes as record data,
// breaking SPF/DKIM/DMARC.
func TestRecordToNativeTXTUnquoted(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, "v=spf1 include:spf.example.net -all")

	entry, err := recordToNative(rc)
	if err != nil {
		t.Fatal(err)
	}
	if want := "v=spf1 include:spf.example.net -all"; entry.Content != want {
		t.Errorf("recordToNative() TXT Content = %q, want %q (no enclosing quotes)", entry.Content, want)
	}

	// Round-trip: reading the value back yields the original.
	got, err := nativeToRecord(entry, dc)
	if err != nil {
		t.Fatal(err)
	}
	if got.GetTargetTXTJoined() != rc.GetTargetTXTJoined() {
		t.Errorf("round-trip TXT = %q, want %q", got.GetTargetTXTJoined(), rc.GetTargetTXTJoined())
	}
}

// TestNativeToRecordHealsLegacyQuotedTXT documents that a TXT value written by
// the earlier buggy code (stored with literal enclosing quotes) is read back
// with those quotes intact, so a subsequent push detects the drift and rewrites
// the record unquoted.
func TestNativeToRecordHealsLegacyQuotedTXT(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	legacy := domain.DNSEntry{Name: "www", Type: "TXT", Content: `"v=spf1 -all"`, Expire: 300}

	got, err := nativeToRecord(legacy, dc)
	if err != nil {
		t.Fatal(err)
	}
	if want := `"v=spf1 -all"`; got.GetTargetTXTJoined() != want {
		t.Errorf("legacy quoted TXT read as %q, want %q (quotes preserved so the diff triggers a rewrite)", got.GetTargetTXTJoined(), want)
	}
}
