package mustbe_test

import (
	"net/netip"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/mustbe"
)

func TestIPv4_RoundTrip(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		a any
	}{
		{"a", "1.2.3.4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert
			original := tt.a
			originalStr := tt.a.(string)
			first, _ := mustbe.IPv4(original)
			firstStr := first.String()
			if firstStr != originalStr {
				t.Errorf("IPv4(%v) = %v, want %v", original, originalStr, firstStr)
			}
			// Round Trip
			second, _ := mustbe.IPv4(firstStr)
			secondStr := second.String()
			if secondStr != originalStr {
				t.Errorf("IPv4(%v) = %v, want %v", original, originalStr, secondStr)
			}
		})
	}
}

func TestIPv4_Parse(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		a    any
		want netip.Addr
	}{
		{"a", "1.2.3.4", netip.MustParseAddr("1.2.3.4")},
		{"b", float64((2 << 24) + (3 << 16) + (4 << 8) + 5), netip.MustParseAddr("2.3.4.5")},
		{"c", netip.MustParseAddr("3.4.5.6"), netip.MustParseAddr("3.4.5.6")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := mustbe.IPv4(tt.a)
			if tt.want != got {
				t.Errorf("IPv4() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPv6(t *testing.T) {
	tests := []struct {
		name        string // description of this test case
		a           any
		shouldError bool
	}{
		{"b", "45::1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert
			original := tt.a
			originalStr := tt.a.(string)
			first, _ := mustbe.IPv6(original)
			firstStr := first.String()
			if firstStr != originalStr {
				t.Errorf("IPv6(%v) = %v, want %v", original, originalStr, firstStr)
			}
			// Round Trip
			second, _ := mustbe.IPv6(firstStr)
			secondStr := second.String()
			if secondStr != originalStr {
				t.Errorf("IPv6(%v) = %v, want %v", original, originalStr, secondStr)
			}
		})
	}
}
