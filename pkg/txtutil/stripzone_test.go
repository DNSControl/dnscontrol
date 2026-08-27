package txtutil_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

func TestStripZone(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		label string
		zone  string
		want  string
	}{
		{"a1", "", "", "@"},
		{"a2", "@", "", "@"},
		{"a3", "*", "", "*"},
		{"a4", "www", "", "www"},
		{"a5", "www.example.com", "", "www.example.com"},
		{"a6", "www.example.com.", "", "www.example.com."},
		{"a7", "www.example.com", "wrong.com", "www.example.com"},
		//
		{"b1", "", "example.com.", "@"},
		{"b2", "@", "example.com.", "@"},
		{"b3", "*", "example.com.", "*"},
		{"b4", "www", "example.com.", "www"},
		{"b5", "www.example.com", "example.com.", "www"},
		{"b6", "www.example.com.", "example.com.", "www"},
		//
		{"c1", "", "example.com", "@"},
		{"c2", "@", "example.com", "@"},
		{"c3", "*", "example.com", "*"},
		{"c4", "www", "example.com", "www"},
		{"c5", "www.example.com", "example.com", "www"},
		{"c6", "www.example.com.", "example.com", "www"},
		//
		{"d1", "www.example.com.", "example.com", "www"},
		{"d2", "www.example.com.", "example.com.", "www"},
		{"d3", "www.example.com.", ".example.com", "www"},
		{"d4", "www.example.com.", ".example.com.", "www"},
		//
		{"e1", "www.example.com", "example.com", "www"},
		{"e2", "www.example.com", "example.com.", "www"},
		{"e3", "www.example.com", ".example.com", "www"},
		{"e4", "www.example.com", ".example.com.", "www"},
		//
		{"f1", "wwwexample.com.", "example.com", "wwwexample.com."},
		{"f2", "example.com.", "example.com", "@"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := txtutil.StripZone(tt.label, tt.zone)
			if got != tt.want {
				t.Errorf("StripZone(%q, %q) = %v, want %v", tt.label, tt.zone, got, tt.want)
			}
		})
	}
}
