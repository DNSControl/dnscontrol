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
			// TODO: update the condition below to compare got with tt.want.
			if got != tt.want {
				t.Errorf("LabelFromFQDNNoDot(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
