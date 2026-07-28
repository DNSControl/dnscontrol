package models_test

import (
	"fmt"
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestSvcbv2ValueToString(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"0 test.com. "},
		{"1 test.com. port=80"},
		{"1 test.com. alpn=h2 port=99"},
		{"3 example.com. alpn=h2,h3 port=999"},
		{"3 example.com. alpn=h2,h3 port=999 ech=some+base64+encoded+value///"},
		{"3 example.com. alpn=h2 port=80 ech=another+base64+encoded+value"},
		{"3 yetanother.com. alpn=h2 port=80 ech=another+base64+encoded+value"},
		{"3 example.com. alpn=h2,h3 port=999"},
		{"1 . "},
		{"2 . alpn=h3,h2 port=443 ipv4hint=123.123.123.123 ipv6hint=dead::beaf"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			rd, _ := dnsv2.NewData(dnsv2.TypeSVCB, tt.input, "example.com")
			want := strings.TrimSpace(tt.input[strings.Index(tt.input, ". ")+2:])
			got := models.Svcbv2ValueToString(rd.(dnsrdatav2.SVCB).Value)
			if got != want {
				t.Errorf("Svcbv2ValueToString() = %q, want %q", got, want)
			}
		})
	}
}
