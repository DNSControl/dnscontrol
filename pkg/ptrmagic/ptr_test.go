package ptrmagic

import (
	"fmt"
	"testing"
)

// ptrmagic(name, domain string, al int) (string, error)

func TestPtrMagic(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		output string
		fail   bool
	}{
		// Magic IPv4:
		{"1.2.3.4", "3.2.1.in-addr.arpa", "4", false},
		{"1.2.3.4", "2.1.in-addr.arpa", "4.3", false},
		{"1.2.3.4", "1.in-addr.arpa", "4.3.2", false},

		// No magic IPv4:
		{"1", "2.3.4.in-addr.arpa", "1", false},
		{"1.2", "3.4.in-addr.arpa", "1.2", false},
		{"1.2.3", "4.in-addr.arpa", "1.2.3", false},
		{"1.2.3.4", "in-addr.arpa", "1.2.3.4", false}, // Not supported, but it works.

		// Magic IPv6:
		{"1", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1", false},
		{"1.0", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0", false},
		{"1.0.0", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0", false},
		{"1.0.0.0", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0", false},
		{"1.0.0.0.0", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0.0", false},
		{"1.0.0.0.0.0", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0.0.0", false},
		{"1.0.0.0.0.0.0", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0.0.0.0", false},
		{"1.0.0.0.0.0.0.0", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0.0.0.0.0", false},
		{"1.0.0.0.0.0.0.0.0", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0.0.0.0.0.0", false},
		{"1.0.0.0.0.0.0.0.0.0", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0.0.0.0.0.0.0", false},
		{"1.0.0.0.0.0.0.0.0.0.0", "0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0.0.0.0.0.0.0.0", false},
		{"1.0.0.0.0.0.0.0.0.0.0.0", "0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0.0.0.0.0.0.0.0.0", false},
		{"1.0.0.0.0.0.0.0.0.0.0.0.0", "0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0.0.0.0.0.0.0.0.0.0", false},
		{"1.0.0.0.0.0.0.0.0.0.0.0.0.0", "0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa", "1.0.0.0.0.0.0.0.0.0.0.0.0.0", false},

		// RFC2317 (Classless)
		// 172.20.18.160/27 is .160 - .191:
		{"172.20.18.159", "160/27.18.20.172.in-addr.arpa", "", true},
		{"172.20.18.160", "160/27.18.20.172.in-addr.arpa", "160", false},
		{"172.20.18.191", "160/27.18.20.172.in-addr.arpa", "191", false},
		{"172.20.18.192", "160/27.18.20.172.in-addr.arpa", "", true},

		// RFC4183 (Classless)
		// 172.20.18.160/27 is .160 - .191:
		{"172.20.18.159", "160-27.18.20.172.in-addr.arpa", "", true},
		{"172.20.18.160", "160-27.18.20.172.in-addr.arpa", "160", false},
		{"172.20.18.191", "160-27.18.20.172.in-addr.arpa", "191", false},
		{"172.20.18.192", "160-27.18.20.172.in-addr.arpa", "", true},
		// 172.20.160.0/19 is .160 - .191:
		{"172.20.159.1", "160-19.20.172.in-addr.arpa", "", true},
		{"172.20.160.2", "160-19.20.172.in-addr.arpa", "2.160", false},
		{"172.20.191.3", "160-19.20.172.in-addr.arpa", "3.191", false},
		{"172.20.192.4", "160-19.20.172.in-addr.arpa", "", true},
		// 172.160.0.0/11 is .160 - .191:
		{"172.159.1.5", "160-11.172.in-addr.arpa", "", true},
		{"172.160.2.6", "160-11.172.in-addr.arpa", "6.2.160", false},
		{"172.191.3.7", "160-11.172.in-addr.arpa", "7.3.191", false},
		{"172.192.4.8", "160-11.172.in-addr.arpa", "", true},

		// If it doesn't end in .arpa, the magic is disabled:
		{"1.2.3.4", "example.com", "1.2.3.4", false},
		{"1", "example.com", "1", false},
		{"1.0.0.0", "example.com", "1.0.0.0", false},
		{"1.0.0.0.0.0.0.0", "example.com", "1.0.0.0.0.0.0.0", false},

		// User manually reversed addresses:
		{"1.1.1.1.in-addr.arpa.", "1.1.in-addr.arpa", "1.1", false},
		{
			"1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.",
			"0.2.ip6.arpa", "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0", false,
		},

		// Error cases:
		{"1.1.1.1.in-addr.arpa.", "2.2.in-addr.arpa", "", true},
		{"1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.", "9.9.ip6.arpa", "", true},
		{"3.3.3.3", "4.4.in-addr.arpa", "", true},
		{"2001:db8::1", "9.9.ip6.arpa", "", true},

		// These should be errors but we don't check for them at this time:
		// {"blurg", "3.4.in-addr.arpa", "blurg", true},
		// {"1", "3.4.in-addr.arpa", "1", true},
	}
	for _, tst := range tests {
		t.Run(fmt.Sprintf("%s %s", tst.name, tst.domain), func(t *testing.T) {
			o, errs := PtrNameMagic(tst.name, tst.domain)
			if errs != nil && !tst.fail {
				t.Errorf("Got error but expected none (%v)", errs)
			} else if errs == nil && tst.fail {
				t.Errorf("Expected error but got none (%v)", o)
			} else if errs == nil && o != tst.output {
				t.Errorf("Got (%v) expected (%v)", o, tst.output)
			}
		})
	}
}
