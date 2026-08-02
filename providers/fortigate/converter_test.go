package fortigate

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestNativeToRecord(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		record     fgDNSRecord
		wantLabel  string
		wantTarget string
	}{
		{"A", fgDNSRecord{Type: "A", Hostname: "www", IP: "192.0.2.1", TTL: 300, Status: "enable"}, "www", "192.0.2.1"},
		{"A", fgDNSRecord{Type: "A", Hostname: "@", IP: "192.0.2.1", TTL: 300, Status: "enable"}, "@", "192.0.2.1"},
		{"A", fgDNSRecord{Type: "A", Hostname: "example.com.", IP: "192.0.2.1", TTL: 300, Status: "enable"}, "@", "192.0.2.1"},
		{"CNAME", fgDNSRecord{Type: "CNAME", Hostname: "www", CanonicalName: "target.example.net.", TTL: 300, Status: "enable"}, "www", "target.example.net."},
		{"MX", fgDNSRecord{Type: "MX", Hostname: "mail.example.com.", Preference: 10, TTL: 300, Status: "disable"}, "@", "10 mail.example.com."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := nativeToRecord(dc, tc.record)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
			if got := rc.GetLabel(); got != tc.wantLabel {
				t.Errorf("label = %q, want %q", got, tc.wantLabel)
			}
		})
	}
}

func TestNativeToRecord_Gen(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		record     fgDNSRecord
		wantTarget string
	}{
		// These were taken from the API directly while running integration tests.
		{"a1", fgDNSRecord{ID: 1, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "1.1.1.1", IPv6: "::", Hostname: "foo"}, "1.1.1.1"},
		{"a2", fgDNSRecord{ID: 2, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "12.12.12.12", IPv6: "::", Hostname: "foo"}, "12.12.12.12"},
		{"a3", fgDNSRecord{ID: 3, Status: "enable", Type: "AAAA", TTL: 300, Preference: 10, IP: "0.0.0.0", IPv6: "2003:dd:d7ff::fe71:aaaa", Hostname: "foo"}, "2003:dd:d7ff::fe71:aaaa"},
		{"a4", fgDNSRecord{ID: 4, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "3.3.3.3", IPv6: "::", Hostname: "zzz"}, "3.3.3.3"},
		{"a5", fgDNSRecord{ID: 5, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "4.4.4.4", IPv6: "::", Hostname: "zzz"}, "4.4.4.4"},
		{"a6", fgDNSRecord{ID: 6, Status: "enable", Type: "AAAA", TTL: 300, Preference: 10, IP: "0.0.0.0", IPv6: "2003:dd:d7ff::fe71:cccc", Hostname: "zzz"}, "2003:dd:d7ff::fe71:cccc"},
		{"b1", fgDNSRecord{ID: 1, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "1.1.1.1", IPv6: "::", Hostname: "foo"}, "1.1.1.1"},
		{"b2", fgDNSRecord{ID: 2, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "12.12.12.12", IPv6: "::", Hostname: "foo"}, "12.12.12.12"},
		{"b3", fgDNSRecord{ID: 3, Status: "enable", Type: "AAAA", TTL: 300, Preference: 10, IP: "0.0.0.0", IPv6: "2003:dd:d7ff::fe71:aaaa", Hostname: "foo"}, "2003:dd:d7ff::fe71:aaaa"},
		{"b4", fgDNSRecord{ID: 4, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "3.3.3.3", IPv6: "::", Hostname: "zzz"}, "3.3.3.3"},
		{"b5", fgDNSRecord{ID: 5, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "4.4.4.4", IPv6: "::", Hostname: "zzz"}, "4.4.4.4"},
		{"b6", fgDNSRecord{ID: 6, Status: "enable", Type: "AAAA", TTL: 300, Preference: 10, IP: "0.0.0.0", IPv6: "2003:dd:d7ff::fe71:cccc", Hostname: "zzz"}, "2003:dd:d7ff::fe71:cccc"},
		{"c1", fgDNSRecord{ID: 1, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "1.1.1.1", IPv6: "::", Hostname: "foo"}, "1.1.1.1"},
		{"c2", fgDNSRecord{ID: 2, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "13.13.13.13", IPv6: "::", Hostname: "foo"}, "13.13.13.13"},
		{"c3", fgDNSRecord{ID: 3, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "3.3.3.3", IPv6: "2003:dd:d7ff::fe71:aaaa", Hostname: "zzz"}, "3.3.3.3"},
		{"c4", fgDNSRecord{ID: 4, Status: "enable", Type: "A", TTL: 300, Preference: 10, IP: "4.4.4.4", IPv6: "::", Hostname: "zzz"}, "4.4.4.4"},
		{"c5", fgDNSRecord{ID: 5, Status: "enable", Type: "AAAA", TTL: 300, Preference: 10, IP: "4.4.4.4", IPv6: "2003:dd:d7ff::fe71:cccc", Hostname: "zzz"}, "2003:dd:d7ff::fe71:cccc"},
		{"c6", fgDNSRecord{ID: 6, Status: "enable", Type: "AAAA", TTL: 300, Preference: 10, IP: "0.0.0.0", IPv6: "2003:dd:d7ff::fe71:aaaa", Hostname: "foo"}, "2003:dd:d7ff::fe71:aaaa"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := nativeToRecord(dc, tc.record)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
		})
	}
}
