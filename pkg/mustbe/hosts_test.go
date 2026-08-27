package mustbe_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

func TestTargetHost(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		origin    string
		arg       any
		want      string
		wantNoDot string
	}{
		// Examples in the docs:
		{"a1", "domain.com", "@", "domain.com.", ""},
		{"a2", "domain.com", "", "domain.com.", ""},
		{"a3", "domain.com", "foo.domain.com.", "foo.domain.com.", ""},
		{"a4", "domain.com", "domain.com.", "domain.com.", ""},
		{"a5", "domain.com", "other.com.", "other.com.", ""},
		{"a6", "domain.com", "short", "short.domain.com.", "short."},
		// Doc examples with name in Unicode:
		{"b1", "domain.com", "München.domain.com.", "xn--mnchen-3ya.domain.com.", ""},
		{"b2", "domain.com", "münchen.domain.com.", "xn--mnchen-3ya.domain.com.", ""},
		{"b3", "domain.com", "München", "xn--mnchen-3ya.domain.com.", "xn--mnchen-3ya."},
		{"b4", "domain.com", "münchen", "xn--mnchen-3ya.domain.com.", "xn--mnchen-3ya."},
		{"b5", "domain.com", "other.com.", "other.com.", ""},
		// Doc examples with origin as Punycode:
		{"c1", "xn--mnchen-3ya.com", "@", "xn--mnchen-3ya.com.", ""},
		{"c2", "xn--mnchen-3ya.com", "", "xn--mnchen-3ya.com.", ""},
		{"c3", "xn--mnchen-3ya.com", "foo.xn--mnchen-3ya.com.", "foo.xn--mnchen-3ya.com.", ""},
		{"c4", "xn--mnchen-3ya.com", "xn--mnchen-3ya.com.", "xn--mnchen-3ya.com.", ""},
		{"c5", "xn--mnchen-3ya.com", "other.com.", "other.com.", ""},
		{"c6", "xn--mnchen-3ya.com", "short", "short.xn--mnchen-3ya.com.", "short."},
		// Doc examples with origin as Punycode and name in Unicode:
		{"d1", "xn--mnchen-3ya.com", "@", "xn--mnchen-3ya.com.", ""},
		{"d2", "xn--mnchen-3ya.com", "", "xn--mnchen-3ya.com.", ""},
		{"d3", "xn--mnchen-3ya.com", "foo.München.com.", "foo.xn--mnchen-3ya.com.", ""},
		{"d4", "xn--mnchen-3ya.com", "München", "xn--mnchen-3ya.xn--mnchen-3ya.com.", "xn--mnchen-3ya."},
		{"d5", "xn--mnchen-3ya.com", "München.com.", "xn--mnchen-3ya.com.", "xn--mnchen-3ya.com."},
		{"d6", "xn--mnchen-3ya.com", "other.com.", "other.com.", ""},
		// Doc examples with orgin and name in PunyCode:
		{"e1", "xn--mnchen-3ya.com", "foo.xn--mnchen-3ya.com.", "foo.xn--mnchen-3ya.com.", ""},
		{"e2", "xn--mnchen-3ya.com", "xn--mnchen-3ya.xn--mnchen-3ya.com.", "xn--mnchen-3ya.xn--mnchen-3ya.com.", ""},
		{"e3", "xn--mnchen-3ya.com", "xn--mnchen-3ya.com.", "xn--mnchen-3ya.com.", ""},
		{"e4", "xn--mnchen-3ya.com", "xn--mnchen-3ya", "xn--mnchen-3ya.xn--mnchen-3ya.com.", "xn--mnchen-3ya."},
		// Other cases:
		//{"f1", "domain.com", "", "domain.com.", ""}, // shouldn't happen.
		{"f2", "domain.com", "42", "42.domain.com.", "42."},
		{"f3", "domain.com", 99, "99.domain.com.", "99."},
		// NullMX and NullSRV:
		{"null", "example.com", ".", ".", "."},
		{"null2", "", ".", ".", "."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mustbe.TargetHost(tt.origin, nrc.Flags{TargetIsFqdnNoDot: false}, tt.arg)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("Host() = %v, want %v", got, tt.want)
			}

			wanted := tt.wantNoDot
			if wanted == "" {
				wanted = tt.want
			}
			got, err = mustbe.TargetHost(tt.origin, nrc.Flags{TargetIsFqdnNoDot: true}, tt.arg)
			if err != nil {
				t.Fatal(err)
			}
			if got != wanted {
				t.Errorf("Host(ENABLED) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTargetHostSingleDotViolation(t *testing.T) {
	for _, targetHost := range []func(string, nrc.Flags, any) (string, error){mustbe.TargetHost, mustbe.TargetHostSRV} {
		got, err := targetHost("example.com", nrc.Flags{EnforceOneDotPolicy: true}, "www.example.com")
		if err == nil {
			t.Fatal("expected single-dot violation error")
		}
		if got != "" {
			t.Errorf("TargetHost() = %q on error, want empty string", got)
		}
		want := `target "www.example.com" must end with a (.) [https://docs.dnscontrol.org/language-reference/why-the-dot]`
		if err.Error() != want {
			t.Errorf("TargetHost() error = %q, want %q", err, want)
		}
	}
}
