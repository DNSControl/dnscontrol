package dnsmadeeasy

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestToRecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	tests := []struct {
		name       string
		record     recordResponseDataEntry
		wantType   string
		wantLabel  string
		wantTarget string
	}{
		{"MX", recordResponseDataEntry{Name: "@", Type: "MX", Value: "mail.example.net.", TTL: 300, MxLevel: 10}, "MX", "@", "10 mail.example.net."},
		{"SRV", recordResponseDataEntry{Name: "_sip._tcp", Type: "SRV", Value: "sip.example.net.", TTL: 300, Priority: 1, Weight: 2, Port: 5060}, "SRV", "_sip._tcp", "1 2 5060 sip.example.net."},
		{"CAA", recordResponseDataEntry{Name: "@", Type: "CAA", Value: `"letsencrypt.org"`, TTL: 300, CaaType: "issue", IssuerCritical: 1}, "CAA", "@", `1 issue "letsencrypt.org"`},
		{"TXT", recordResponseDataEntry{Name: "@", Type: "TXT", Value: `"quoted text"`, TTL: 300}, "TXT", "@", `"quoted text"`},
		{"ANAME", recordResponseDataEntry{Name: "@", Type: "ANAME", Value: "target.example.net.", TTL: 300}, "ALIAS", "@", "target.example.net."},
		// The API returns an empty name for records at the zone apex.
		{"apex empty name", recordResponseDataEntry{Name: "", Type: "A", Value: "1.2.3.4", TTL: 300}, "A", "@", "1.2.3.4"},
		{"mixed case name", recordResponseDataEntry{Name: "WWW", Type: "A", Value: "1.2.3.4", TTL: 300}, "A", "www", "1.2.3.4"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.record
			rc := toRecordConfig(dc, &record)
			if got := rc.Type; got != tc.wantType {
				t.Errorf("type = %q, want %q", got, tc.wantType)
			}
			if got := rc.GetLabel(); got != tc.wantLabel {
				t.Errorf("label = %q, want %q", got, tc.wantLabel)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
			if got := rc.TTL; got != uint32(tc.record.TTL) {
				t.Errorf("ttl = %d, want %d", got, tc.record.TTL)
			}
			if rc.Original != &record {
				t.Error("original provider record was not retained")
			}
		})
	}
}

// TestApexRoundTrip checks both halves of the apex label mapping: the API sends
// an empty name for apex records, and expects one back.
func TestApexRoundTrip(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc := toRecordConfig(dc, &recordResponseDataEntry{Name: "", Type: "A", Value: "1.2.3.4", TTL: 300})
	if got := rc.GetLabelFQDN(); got != "example.com" {
		t.Errorf("fqdn = %q, want %q", got, "example.com")
	}
	if got := fromRecordConfig(rc).Name; got != "" {
		t.Errorf("name = %q, want %q", got, "")
	}
}

func TestSystemNameServerToRecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc := systemNameServerToRecordConfig(dc, "ns1.example.net")
	if got := rc.Type; got != "NS" {
		t.Errorf("type = %q, want %q", got, "NS")
	}
	if got := rc.GetLabel(); got != "@" {
		t.Errorf("label = %q, want %q", got, "@")
	}
	if got := rc.GetLabelFQDN(); got != "example.com" {
		t.Errorf("fqdn = %q, want %q", got, "example.com")
	}
	if got := rc.GetRDATA().String(); got != "ns1.example.net." {
		t.Errorf("target = %q, want %q", got, "ns1.example.net.")
	}
	if got := rc.TTL; got != fixedNameServerRecordTTL {
		t.Errorf("ttl = %d, want %d", got, fixedNameServerRecordTTL)
	}
}
