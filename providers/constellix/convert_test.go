package constellix

import (
	"encoding/json"
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

// record builds an API record object of example.com from its JSON value list.
func record(name, rtype string, ttl uint32, values string) apiRecord {
	return apiRecord{ID: "7", Name: name, Type: rtype, TTL: ttl, Mode: "standard", Region: "default", Value: json.RawMessage(values)}
}

func TestToRC(t *testing.T) {
	dc := models.MustNewDomainConfig(testZone)
	long := strings.Repeat("a", 300)

	tests := []struct {
		name  string
		rec   apiRecord
		label string
		// want are the records as they would be configured in dnsconfig.js.
		want []*models.RecordConfig
	}{
		{
			name:  "apex A with two values",
			rec:   record("", "A", 300, `[{"value": "192.0.2.1", "enabled": true}, {"value": "192.0.2.2", "enabled": true}]`),
			label: "@",
			want: []*models.RecordConfig{
				dc.MustNewRecordConfig("@", 300, dnsv2.TypeA, "192.0.2.1"),
				dc.MustNewRecordConfig("@", 300, dnsv2.TypeA, "192.0.2.2"),
			},
		},
		{
			name:  "expanded AAAA",
			rec:   record("www", "AAAA", 300, `[{"value": "2001:db8:0:0:0:0:0:1", "enabled": true}]`),
			label: "www",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("www", 300, dnsv2.TypeAAAA, "2001:db8::1")},
		},
		{
			name:  "CNAME with an absolute target",
			rec:   record("cn", "CNAME", 300, `[{"value": "target.example.net.", "enabled": true}]`),
			label: "cn",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("cn", 300, dnsv2.TypeCNAME, "target.example.net.")},
		},
		{
			name:  "CNAME with a relative target",
			rec:   record("cnrel", "CNAME", 300, `[{"value": "www", "enabled": true}]`),
			label: "cnrel",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("cnrel", 300, dnsv2.TypeCNAME, "www.example.com.")},
		},
		{
			name:  "ANAME is ALIAS",
			rec:   record("", "ANAME", 300, `[{"value": "dns.google.", "enabled": true}]`),
			label: "@",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("@", 300, privatetypes.TypeALIAS, "dns.google.")},
		},
		{
			name:  "MX with absolute and relative servers",
			rec:   record("", "MX", 300, `[{"server": "mx1.example.net.", "priority": 10, "enabled": true}, {"server": "mx2", "priority": 20, "enabled": true}]`),
			label: "@",
			want: []*models.RecordConfig{
				dc.MustNewRecordConfig("@", 300, dnsv2.TypeMX, uint16(10), "mx1.example.net."),
				dc.MustNewRecordConfig("@", 300, dnsv2.TypeMX, uint16(20), "mx2.example.com."),
			},
		},
		{
			name:  "SRV",
			rec:   record("_sip._tcp", "SRV", 300, `[{"priority": 10, "weight": 5, "port": 5060, "host": "sip.example.net.", "enabled": true}]`),
			label: "_sip._tcp",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("_sip._tcp", 300, dnsv2.TypeSRV, uint16(10), uint16(5), uint16(5060), "sip.example.net.")},
		},
		{
			name:  "CAA",
			rec:   record("", "CAA", 300, `[{"tag": "issue", "data": "letsencrypt.org", "flags": 0, "enabled": true}, {"tag": "iodef", "data": "mailto:a@example.net", "flags": 128, "enabled": true}]`),
			label: "@",
			want: []*models.RecordConfig{
				dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(0), "issue", "letsencrypt.org"),
				dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(128), "iodef", "mailto:a@example.net"),
			},
		},
		{
			name:  "NS",
			rec:   record("sub", "NS", 300, `[{"host": "ns1.example.net.", "enabled": true}]`),
			label: "sub",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("sub", 300, dnsv2.TypeNS, "ns1.example.net.")},
		},
		{
			name:  "PTR",
			rec:   record("4", "PTR", 300, `[{"system": "host.example.net.", "enabled": true}]`),
			label: "4",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("4", 300, dnsv2.TypePTR, "host.example.net.")},
		},
		{
			name:  "TXT",
			rec:   record("", "TXT", 300, `[{"value": "\"v=spf1 -all\"", "enabled": true}]`),
			label: "@",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("@", 300, dnsv2.TypeTXT, "v=spf1 -all")},
		},
		{
			name:  "TXT with escaped quotes and backslash",
			rec:   record("t", "TXT", 300, `[{"value": "\"say \\\"hi\\\" and \\\\ back\"", "enabled": true}]`),
			label: "t",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("t", 300, dnsv2.TypeTXT, `say "hi" and \ back`)},
		},
		{
			name:  "long TXT stored as two strings",
			rec:   record("t", "TXT", 300, `[{"value": "\"`+long[:255]+`\"\"`+long[255:]+`\"", "enabled": true}]`),
			label: "t",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("t", 300, dnsv2.TypeTXT, long)},
		},
		{
			// The API splits a long value by its escaped length, so the
			// strings break at another place than DNSControl would.
			name:  "long TXT split inside escapes",
			rec:   record("t", "TXT", 300, `[{"value": "\"`+strings.Repeat("A", 250)+`x\\\"y\"\"\\\\z\\195\\169`+strings.Repeat("B", 60)+`\"", "enabled": true}]`),
			label: "t",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("t", 300, dnsv2.TypeTXT, strings.Repeat("A", 250)+`x"y\zé`+strings.Repeat("B", 60))},
		},
		{
			name:  "TXT with an escaped non-ASCII character",
			rec:   record("t", "TXT", 300, `[{"value": "\"caf\\195\\169\"", "enabled": true}]`),
			label: "t",
			want:  []*models.RecordConfig{dc.MustNewRecordConfig("t", 300, dnsv2.TypeTXT, "café")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toRC(dc, tt.rec)
			if err != nil {
				t.Fatalf("toRC() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("toRC() returned %d records, want %d", len(got), len(tt.want))
			}
			for i, rc := range got {
				if rc.GetLabel() != tt.label || rc.TTL != tt.rec.TTL || rc.Type != tt.want[i].Type {
					t.Errorf("record %d = %s %d %s, want %s %d %s", i, rc.GetLabel(), rc.TTL, rc.Type, tt.label, tt.rec.TTL, tt.want[i].Type)
				}
				if rc.ComparableV3 != tt.want[i].ComparableV3 {
					t.Errorf("record %d = %q, want %q", i, rc.ComparableV3, tt.want[i].ComparableV3)
				}
				if rc.Original.(apiRecord).ID != "7" {
					t.Errorf("record %d does not keep the API record as Original", i)
				}
			}
		})
	}
}

func TestToNative(t *testing.T) {
	dc := models.MustNewDomainConfig(testZone)
	long := strings.Repeat("a", 300)

	tests := []struct {
		name    string
		records models.Records
		want    string
	}{
		{
			name: "apex A with two values",
			records: models.Records{
				dc.MustNewRecordConfig("@", 300, dnsv2.TypeA, "192.0.2.1"),
				dc.MustNewRecordConfig("@", 300, dnsv2.TypeA, "192.0.2.2"),
			},
			want: `{"name":"","type":"A","ttl":300,"value":[{"enabled":true,"value":"192.0.2.1"},{"enabled":true,"value":"192.0.2.2"}]}`,
		},
		{
			name:    "AAAA",
			records: models.Records{dc.MustNewRecordConfig("www", 300, dnsv2.TypeAAAA, "2001:db8::1")},
			want:    `{"name":"www","type":"AAAA","ttl":300,"value":[{"enabled":true,"value":"2001:db8::1"}]}`,
		},
		{
			name:    "CNAME",
			records: models.Records{dc.MustNewRecordConfig("cn", 300, dnsv2.TypeCNAME, "www")},
			want:    `{"name":"cn","type":"CNAME","ttl":300,"value":[{"enabled":true,"value":"www.example.com."}]}`,
		},
		{
			name:    "ALIAS",
			records: models.Records{dc.MustNewRecordConfig("@", 300, privatetypes.TypeALIAS, "foo")},
			want:    `{"name":"","type":"ANAME","ttl":300,"value":[{"enabled":true,"value":"foo.example.com."}],"skipLookup":true}`,
		},
		{
			name:    "MX",
			records: models.Records{dc.MustNewRecordConfig("@", 300, dnsv2.TypeMX, uint16(10), "mx1.example.net.")},
			want:    `{"name":"","type":"MX","ttl":300,"value":[{"enabled":true,"priority":10,"server":"mx1.example.net."}]}`,
		},
		{
			name:    "SRV",
			records: models.Records{dc.MustNewRecordConfig("_sip._tcp", 300, dnsv2.TypeSRV, uint16(10), uint16(5), uint16(5060), "sip")},
			want:    `{"name":"_sip._tcp","type":"SRV","ttl":300,"value":[{"enabled":true,"host":"sip.example.com.","port":5060,"priority":10,"weight":5}]}`,
		},
		{
			name:    "CAA",
			records: models.Records{dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, uint8(128), "iodef", "mailto:a@example.net")},
			want:    `{"name":"","type":"CAA","ttl":300,"value":[{"data":"mailto:a@example.net","enabled":true,"flags":128,"tag":"iodef"}]}`,
		},
		{
			name:    "NS",
			records: models.Records{dc.MustNewRecordConfig("sub", 300, dnsv2.TypeNS, "ns1.example.net.")},
			want:    `{"name":"sub","type":"NS","ttl":300,"value":[{"enabled":true,"host":"ns1.example.net."}]}`,
		},
		{
			name:    "PTR",
			records: models.Records{dc.MustNewRecordConfig("4", 300, dnsv2.TypePTR, "host.example.net.")},
			want:    `{"name":"4","type":"PTR","ttl":300,"value":[{"enabled":true,"system":"host.example.net."}]}`,
		},
		{
			name:    "TXT with quotes, backslash and non-ASCII",
			records: models.Records{dc.MustNewRecordConfig("t", 300, dnsv2.TypeTXT, `say "hi" \ café`)},
			want:    `{"name":"t","type":"TXT","ttl":300,"value":[{"enabled":true,"value":"\"say \\\"hi\\\" \\\\ caf\\195\\169\""}]}`,
		},
		{
			name:    "long TXT",
			records: models.Records{dc.MustNewRecordConfig("t", 300, dnsv2.TypeTXT, long)},
			want:    `{"name":"t","type":"TXT","ttl":300,"value":[{"enabled":true,"value":"\"` + long[:255] + `\"\"` + long[255:] + `\""}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := toNative(tt.records)
			if err != nil {
				t.Fatal(err)
			}
			got, _ := json.Marshal(req)
			if string(got) != tt.want {
				t.Errorf("toNative() =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}
