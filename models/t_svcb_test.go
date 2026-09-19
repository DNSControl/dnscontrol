package models_test

import (
	"fmt"
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
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
		{"1 . alpn=h3,h2 no-default-alpn"},
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

func TestMakeSVCBKeysWithoutValue(t *testing.T) {
	tests := []struct {
		params  string
		want    string
		wantErr bool
	}{
		{params: "no-default-alpn alpn=h3,h2", want: "no-default-alpn alpn=h3,h2"},
		{params: `alpn=h3,h2 no-default-alpn=""`, want: "alpn=h3,h2 no-default-alpn"},
		{params: "alpn=h3,h2 no-default-alpn=", want: "alpn=h3,h2 no-default-alpn"},
		{params: "ohttp", want: "ohttp"},
		{params: "alpn", wantErr: true},
		{params: "alpn=h2 port", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.params, func(t *testing.T) {
			rd, err := models.MakeSVCB("example.com", nil, nrc.Flags{}, 1, ".", tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("MakeSVCB(%q) expected an error", tt.params)
				}
				return
			}
			if err != nil {
				t.Fatalf("MakeSVCB(%q) error: %v", tt.params, err)
			}
			got := models.Svcbv2ValueToString(rd.(dnsrdatav2.SVCB).Value)
			if got != tt.want {
				t.Errorf("MakeSVCB(%q) = %q, want %q", tt.params, got, tt.want)
			}
		})
	}
}
