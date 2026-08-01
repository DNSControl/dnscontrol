package ovh

import (
	"reflect"
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/ovh/go-ovh/ovh"
)

func Test_getOVHEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{
			"default to EU", "", ovh.OvhEU,
		},
		{
			"default to EU if omitted", "omitted", ovh.OvhEU,
		},
		{
			"set to EU", "eu", ovh.OvhEU,
		},
		{
			"set to CA", "ca", ovh.OvhCA,
		},
		{
			"set to US", "us", ovh.OvhUS,
		},
		{
			"case insensitive", "Eu", ovh.OvhEU,
		},
		{
			"case insensitive ca", "CA", ovh.OvhCA,
		},
		{
			"arbitratry", "https://blah", "https://blah",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := make(map[string]string)
			if tt.endpoint != "" && tt.endpoint != "omitted" {
				params["endpoint"] = tt.endpoint
			}
			if got := getOVHEndpoint(params); got != tt.want {
				t.Errorf("getOVHEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nativeToRecord(t *testing.T) {
	dc := &models.DomainConfig{Name: "dnscontrol.ovh"}

	tests := []struct {
		name       string
		record     *Record
		wantTarget string
		wantErr    bool
	}{
		{
			// Regular TXT records are returned by OVH properly quoted, as
			// RFC1035 zone-file presentation strings.
			name: "TXT quoted, with semicolon",
			record: &Record{
				Target:    `"with a ; semicolon"`,
				Zone:      "dnscontrol.ovh",
				FieldType: "TXT",
				SubDomain: "foo",
			},
			wantTarget: "with a ; semicolon",
		},
		{
			// DMARC records are returned raw/unquoted. A naive zone-file
			// parse would treat the unescaped ';' as a comment and truncate
			// the value at "v=DMARC1".
			name: "native DMARC, unquoted with semicolons",
			record: &Record{
				Target:    "v=DMARC1; p=none; rua=mailto:dmarc@yourdomain.com",
				Zone:      "dnscontrol.ovh",
				FieldType: "DMARC",
				SubDomain: "_dmarc",
			},
			wantTarget: "v=DMARC1; p=none; rua=mailto:dmarc@yourdomain.com",
		},
		{
			// Same issue for DKIM records.
			name: "native DKIM, unquoted with semicolons",
			record: &Record{
				Target:    "v=DKIM1; t=s; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCzwOUg",
				Zone:      "dnscontrol.ovh",
				FieldType: "DKIM",
				SubDomain: "dkim._domainkey",
			},
			wantTarget: "v=DKIM1; t=s; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCzwOUg",
		},
		{
			// Native SPF records, unlike DKIM/DMARC, are returned quoted
			// like a regular TXT record.
			name: "native SPF, quoted",
			record: &Record{
				Target:    `"v=spf1 ip4:99.99.99.99 -all"`,
				Zone:      "dnscontrol.ovh",
				FieldType: "SPF",
				SubDomain: "spf",
			},
			wantTarget: "v=spf1 ip4:99.99.99.99 -all",
		},
		{
			// Real DKIM keys routinely exceed the 255-octet RFC1035
			// character-string limit (e.g. a 2048-bit RSA key). The raw,
			// unquoted target must round-trip in full: NewRecordConfig
			// segments it into 255-octet chunks internally, but joining it
			// back for comparison/display must reproduce the exact input.
			name: "native DKIM, longer than 255 bytes",
			record: &Record{
				Target:    "v=DKIM1; t=s; p=" + strings.Repeat("A", 300),
				Zone:      "dnscontrol.ovh",
				FieldType: "DKIM",
				SubDomain: "dkim._domainkey",
			},
			wantTarget: "v=DKIM1; t=s; p=" + strings.Repeat("A", 300),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nativeToRecord(tt.record, dc)
			if (err != nil) != tt.wantErr {
				t.Errorf("nativeToRecord() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			want := dc.MustNewRecordConfig(dc.LabelFromShort(tt.record.SubDomain), 3600, dnsv2.TypeTXT, tt.wantTarget)
			want.Original = tt.record

			if !reflect.DeepEqual(got, want) {
				t.Errorf("nativeToRecord() got = %#v, want %#v", got, want)
			}
		})
	}
}
