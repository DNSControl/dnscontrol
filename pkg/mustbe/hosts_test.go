package mustbe_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

func TestTargetHost(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		origin string
		arg    any
		want   string
	}{
		// Examples in the docs:
		{"a", "domain.com", "@", "domain.com."},
		{"b", "domain.com", "", "domain.com."},
		{"c", "domain.com", "foo.domain.com.", "foo.domain.com."},
		{"d", "domain.com", "domain.com.", "domain.com."},
		{"e", "domain.com", "other.com.", "other.com."},
		{"f", "domain.com", "short", "short.domain.com."},
		// Doc examples with name in Unicode:
		{"f", "domain.com", "München.domain.com.", "xn--mnchen-3ya.domain.com."},
		{"g", "domain.com", "münchen.domain.com.", "xn--mnchen-3ya.domain.com."},
		{"h", "domain.com", "München", "xn--mnchen-3ya.domain.com."},
		{"i", "domain.com", "münchen", "xn--mnchen-3ya.domain.com."},
		{"j", "domain.com", "other.com.", "other.com."},
		// Doc examples with origin as Punycode:
		{"k", "xn--mnchen-3ya.com", "@", "xn--mnchen-3ya.com."},
		{"k", "xn--mnchen-3ya.com", "", "xn--mnchen-3ya.com."},
		{"m", "xn--mnchen-3ya.com", "foo.xn--mnchen-3ya.com.", "foo.xn--mnchen-3ya.com."},
		{"l", "xn--mnchen-3ya.com", "xn--mnchen-3ya.com.", "xn--mnchen-3ya.com."},
		{"o", "xn--mnchen-3ya.com", "other.com.", "other.com."},
		{"n", "xn--mnchen-3ya.com", "short", "short.xn--mnchen-3ya.com."},
		// Doc examples with origin as Punycode and name in Unicode:
		{"p", "xn--mnchen-3ya.com", "@", "xn--mnchen-3ya.com."},
		{"p", "xn--mnchen-3ya.com", "", "xn--mnchen-3ya.com."},
		{"q", "xn--mnchen-3ya.com", "foo.München.com.", "foo.xn--mnchen-3ya.com."},
		{"r", "xn--mnchen-3ya.com", "München", "xn--mnchen-3ya.xn--mnchen-3ya.com."},
		{"p", "xn--mnchen-3ya.com", "München.com.", "xn--mnchen-3ya.com."},
		{"s", "xn--mnchen-3ya.com", "other.com.", "other.com."},
		// Doc examples with orgin and name in PunyCode:
		{"u", "xn--mnchen-3ya.com", "foo.xn--mnchen-3ya.com.", "foo.xn--mnchen-3ya.com."},
		{"u", "xn--mnchen-3ya.com", "xn--mnchen-3ya.xn--mnchen-3ya.com.", "xn--mnchen-3ya.xn--mnchen-3ya.com."},
		{"t", "xn--mnchen-3ya.com", "xn--mnchen-3ya.com.", "xn--mnchen-3ya.com."},
		{"v", "xn--mnchen-3ya.com", "xn--mnchen-3ya", "xn--mnchen-3ya.xn--mnchen-3ya.com."},
		// Other cases:
		{"w", "domain.com", "", "domain.com."},
		{"x", "domain.com", "42", "42.domain.com."},
		{"y", "domain.com", 99, "99.domain.com."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustbe.TargetHost(tt.origin, tt.arg)
			if got != tt.want {
				t.Errorf("Host() = %v, want %v", got, tt.want)
			}
		})
	}
}
