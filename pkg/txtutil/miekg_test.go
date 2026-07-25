package txtutil_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

func TestZoneifyQuoted(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		txt  []string
		want string
	}{
		{"simple", []string{`simple`}, `"simple"`},
		{"example", []string{`one`, `two`, `3`, `f&&r`, `f ve`}, `"one" "two" "3" "f&&r" "f ve"`},
		{"fqdn", []string{`example.com.`}, `"example.com."`},
		{"space", []string{`with space`}, `"with space"`},
		{"quote", []string{`with'quote`}, `"with'quote"`},
		{"dquote", []string{`with"dquote`}, `"with\"dquote"`},
		// {"backslash", []string{`with\backslash`}, `"with\\backslash"`}, // FAILING
		{"multiple", []string{`line1`, `line2`}, `"line1" "line2"`},
		//{"complex", []string{`line with "dquotes" and \backslash\`}, `"line with "dquote" and \backslash\`}, // FAILING
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := txtutil.ZoneifyQuoted(tt.txt)
			if got != tt.want {
				t.Errorf("ZoneifyQuoted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZoneify(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		txt  []string
		want string
	}{
		{"simple", []string{`simple`}, `simple`},
		{"fqdn", []string{`example.com.`}, `example.com.`},
		{"example", []string{`one`, `two`, `3`, `f&&r`, `f ve`}, `one two 3 "f&&r" "f ve"`},
		{"dots", []string{`do.ts`}, `do.ts`},
		{"at", []string{`@`}, `@`},
		{"wild", []string{`*`}, `*`},
		{"space", []string{`with space`}, `"with space"`},
		{"quote", []string{`with'quote`}, `"with'quote"`},
		{"dquote", []string{`with"dquote`}, `"with\"dquote"`},
		//{"backslash", []string{`with\backslash`}, `"with\\backslash"`},  // FAILING
		{"multiple", []string{`line1`, `line2`}, `line1 line2`},
		{"justone", []string{`line1`, `li}ne2`}, `line1 "li}ne2"`},
		{"empty", []string{``}, `""`},
		//{"complex", []string{`line with "dquotes" and \backslash\`}, `"line with "dquote" and \backslash\`}, // FAILING
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := txtutil.Zoneify(tt.txt)
			if got != tt.want {
				t.Errorf("Zoneify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZoneifyString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"plain", "simple", "simple"},
		{"space", "with space", `"with space"`},
		{"dquote", `with"dquote`, `"with\"dquote"`},
		{"empty", "", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := txtutil.ZoneifyString(tt.s)
			if got != tt.want {
				t.Errorf("ZoneifyString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZoneifyManyAny(t *testing.T) {
	tests := []struct {
		name string
		args []any
		want string
	}{
		{"strings", []any{"one", "two"}, "one two"},
		{"mixed", []any{"a", 2, 3.5, true}, "a 2 3.5 true"},
		{"empty", []any{""}, `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := txtutil.ZoneifyManyAny(tt.args)
			if got != tt.want {
				t.Errorf("ZoneifyManyAny() = %v, want %v", got, tt.want)
			}
		})
	}
}
