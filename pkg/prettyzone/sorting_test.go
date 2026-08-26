package prettyzone_test

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/prettyzone"
)

func TestLabelLess(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		a    string
		b    string
		want bool
	}{
		{"a", "example.com", "example.com", false},
		{"b", "example.com", "foo.example.com", true},
		{"c", "foo.example.com", "example.com", false},
		{"d", "foo.example.com", "bar.example.com", false},
		{"e", "bar.example.com", "foo.example.com", true},
		{"f", "4.5.example.com", "example.com", false},
		{"h", "*.bzt.mup", "aaa.bzt.mup", true},
		{"i", "aaa.bzt.mup", "*.bzt.mup", false},
		{"j", "1.bzt.mup", "aaa.bzt.mup", true},
		{"k", "aaa.bzt.mup", "1.bzt.mup", false},
		{"m", "@", "@", false},
		{"n", "@", "*", true},
		{"o", "*", "@", false},
		{"p", "@", "b", true},
		{"q", "b", "@", false},
		{"v", "*", "b", true},
		{"w", "b", "*", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prettyzone.LabelLess(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("LabelLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestZoneGenData_Less(t *testing.T) {
	dc, _ := models.NewDomainConfig("example.com")
	dc.AddTestRC(t, dc.LabelFromFQDNNoDot("example.com"), 300, dnsv2.TypeRP, "user.example.com.", "example.com.")
	dc.AddTestRC(t, dc.LabelFromFQDNNoDot("4.5.example.com"), 300, dnsv2.TypePTR, "y.bosun.org.")

	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		records    models.Records
		origin     string
		defaultTTL uint32
		comments   []string
		// Named input parameters for target function.
		i    int
		j    int
		want bool
	}{
		{"a", dc.Records, "example.com", 300, nil, 0, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := prettyzone.PrettySort(tt.records, tt.origin, tt.defaultTTL, tt.comments)
			got := z.Less(tt.i, tt.j)
			if got != tt.want {
				t.Errorf("Less(%d, %d) = %v, want %v", tt.i, tt.j, got, tt.want)
			}
		})
	}
}
