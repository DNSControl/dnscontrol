package models_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestDomainConfig_LabelFromFQDNNoDot(t *testing.T) {
	tests := []struct {
		tname      string // description of this test case
		domainname string
		name       string
		want       string
	}{
		{"b", "foo.com", "foo.com", "@"},
		{"a", "foo.com", "bar.foo.com", "bar"},
		{"a", "foo.com", "bat.bar.foo.com", "bat.bar"},
	}
	for _, tt := range tests {
		t.Run(tt.tname, func(t *testing.T) {
			dc, err := models.NewDomainConfig(tt.domainname)
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			got := dc.LabelFromFQDNNoDot(tt.name)
			if got != tt.want {
				t.Errorf("LabelFromFQDNNoDot(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestDomainConfig_LabelFromDnsconfigjs(t *testing.T) {
	const ip6Zone = "8.b.d.0.1.0.0.2.ip6.arpa"
	const ip6Sub = "d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0"
	const nibbleSub = "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0"

	tests := []struct {
		tname      string
		domainname string
		subdomain  string
		label      string
		want       string
	}{
		{"ip6 apex", ip6Zone, ip6Sub, "@", ip6Sub},
		{"ip6 bare address is left alone", ip6Zone, ip6Sub, "2001:db8::abcd", "2001:db8::abcd"},
		{"ip6 two addresses do not collide", ip6Zone, ip6Sub, "2001:db8::abce", "2001:db8::abce"},
		{"ip6 other label joins the subdomain", ip6Zone, nibbleSub, "www", "www." + nibbleSub},
		{"in-addr apex", "3.2.1.in-addr.arpa", "1", "@", "1"},
		{"in-addr short label joins the subdomain", "3.2.1.in-addr.arpa", "1", "4", "4.1"},
		{"in-addr dotted quad is left alone", "3.2.1.in-addr.arpa", "1", "1.2.3.4", "1.2.3.4"},
		{"forward zone label joins the subdomain", "example.com", "sub", "www", "www.sub"},
		{"in-addr zone suffix is stripped", "3.2.1.in-addr.arpa", "", "4.3.2.1.in-addr.arpa", "4"},
		{"in-addr zone suffix is stripped under a subdomain", "8.9.in-addr.arpa", "7", "6.7.8.9.in-addr.arpa", "6.7"},
		{"in-addr short label is unchanged", "3.2.1.in-addr.arpa", "", "4", "4"},
		{"ip6 zone suffix is stripped", ip6Zone, "", "1.0." + ip6Zone, "1.0"},
		{"forward zone suffix is not stripped", "example.com", "", "www.example.com", "www.example.com"},
		{"ip6 zone suffix is stripped under subdomain a", ip6Zone, "a", "1.a." + ip6Zone, "1.a"},
	}
	for _, tt := range tests {
		t.Run(tt.tname, func(t *testing.T) {
			dc, err := models.NewDomainConfig(tt.domainname)
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			got, err := dc.LabelFromDnsconfigjs(tt.label, tt.subdomain)
			if err != nil {
				t.Fatalf("LabelFromDnsconfigjs(%q, %q) returned error: %v", tt.label, tt.subdomain, err)
			}
			if got != tt.want {
				t.Errorf("LabelFromDnsconfigjs(%q, %q) = %v, want %v", tt.label, tt.subdomain, got, tt.want)
			}
		})
	}
}
