package mustbe_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

func TestIPv4(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		a           any
		shouldError bool
	}{
		{"a", "1.2.3.4", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert
			original := tt.a
			originalStr := tt.a.(string)
			first := mustbe.IPv4(original)
			firstStr := first.String()
			if firstStr != originalStr {
				t.Errorf("IPv4(%v) = %v, want %v", original, originalStr, firstStr)
			}
			// Round Trip
			second := mustbe.IPv4(firstStr)
			secondStr := second.String()
			if secondStr != originalStr {
				t.Errorf("IPv4(%v) = %v, want %v", original, originalStr, secondStr)
			}
		})
	}
}

func TestIPv6(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
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
			first := mustbe.IPv6(original)
			firstStr := first.String()
			if firstStr != originalStr {
				t.Errorf("IPv6(%v) = %v, want %v", original, originalStr, firstStr)
			}
			// Round Trip
			second := mustbe.IPv6(firstStr)
			secondStr := second.String()
			if secondStr != originalStr {
				t.Errorf("IPv6(%v) = %v, want %v", original, originalStr, secondStr)
			}
		})
	}
}
