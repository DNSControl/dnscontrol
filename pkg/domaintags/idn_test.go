package domaintags_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/domaintags"
)

func TestEfficientToASCII(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		iname string
		want  string
	}{
		{name: "simple", iname: `foo`, want: `foo`},
		{name: "case", iname: `SIMPLE`, want: `simple`},
		{name: "mixed", iname: `SIMекамплеPLE`, want: `xn--simple-5nf3bb2cml3b`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domaintags.EfficientToASCII(tt.iname)
			if got != tt.want {
				t.Errorf("EfficientToASCII(%q) = %q, want %q", tt.iname, got, tt.want)
			}
		})
	}
}
