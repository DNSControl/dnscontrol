package mustbe_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

func TestRRType(t *testing.T) {
	tests := []struct {
		arg     string
		want    string
		isError bool
	}{
		{"A", "A", false},
		{"aaaa", "AAAA", false},
		{"", "", true},
		{"BOGUS", "BOGUS", true},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			got, err := mustbe.RRType("", nrc.Flags{StrictParamChecking: true}, tt.arg)
			if (err != nil) != tt.isError {
				t.Errorf("strict: got err=%v, wantErr=%v", err, tt.isError)
			}
			if err == nil && got != tt.want {
				t.Errorf("strict: got %q, want %q", got, tt.want)
			}
			// Without the flag, provider data is accepted as-is.
			got, err = mustbe.RRType("", nrc.Flags{}, tt.arg)
			if err != nil || got != tt.want {
				t.Errorf("unchecked: got (%q, %v), want (%q, nil)", got, err, tt.want)
			}
		})
	}
}
