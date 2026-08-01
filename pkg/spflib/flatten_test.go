package spflib

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type fakeResolver map[string]string

func (r fakeResolver) GetSPF(name string) (string, error) {
	spf, ok := r[name]
	if !ok {
		return "", fmt.Errorf("%s has no SPF record", name)
	}
	return spf, nil
}

func TestFlatten(t *testing.T) {
	res, err := NewCache("testdata-dns1.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := Parse(strings.Join([]string{"v=spf1",
		"ip4:198.252.206.0/24",
		"ip4:192.111.0.0/24",
		"include:_spf.google.com",
		"include:mailgun.org",
		//"include:spf-basic.fogcreek.com",
		"include:mail.zendesk.com",
		"include:servers.mcsv.net",
		"include:sendgrid.net",
		"include:spf.mtasv.net",
		"~all"}, " "), res)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rec.Print())
	rec = rec.Flatten("mailgun.org")
	t.Log(rec.Print())
}

func TestFlattenTrailingAll(t *testing.T) {
	tests := []struct {
		description string
		dnsres      fakeResolver
		input       string
		want        string
	}{
		{
			description: "include of a redirect-only target keeps the last netblock",
			dnsres: fakeResolver{
				"aspmx.googlemail.com": "v=spf1 redirect=_spf.google.com",
				"_spf.google.com":      "v=spf1 ip4:74.125.0.0/16 ip4:209.85.128.0/17 ip6:2001:4860:4864::/56 ip6:2404:6800:4864::/56 ip6:2607:f8b0:4864::/56 ip6:2800:3f0:4864::/56 ip6:2a00:1450:4864::/56 ip6:2c0f:fb50:4864::/56 ~all",
			},
			input: "v=spf1 include:aspmx.googlemail.com -all",
			want:  "v=spf1 ip4:74.125.0.0/16 ip4:209.85.128.0/17 ip6:2001:4860:4864::/56 ip6:2404:6800:4864::/56 ip6:2607:f8b0:4864::/56 ip6:2800:3f0:4864::/56 ip6:2a00:1450:4864::/56 ip6:2c0f:fb50:4864::/56 -all",
		},
		{
			description: "include with no all keeps its last term",
			dnsres:      fakeResolver{"spf-00123c01.pphosted.com": "v=spf1 ip4:67.231.153.0/24 ip4:67.231.145.0/24 ip4:67.231.149.0/24 ip4:67.231.152.48 ip4:67.231.148.205 ip4:67.231.152.182"},
			input:       "v=spf1 include:spf-00123c01.pphosted.com -all",
			want:        "v=spf1 ip4:67.231.153.0/24 ip4:67.231.145.0/24 ip4:67.231.149.0/24 ip4:67.231.152.48 ip4:67.231.148.205 ip4:67.231.152.182 -all",
		},
		{
			description: "include ending in ~all drops the all",
			dnsres:      fakeResolver{"example.net": "v=spf1 ip4:1.2.3.4 ip4:5.6.7.8 ~all"},
			input:       "v=spf1 include:example.net -all",
			want:        "v=spf1 ip4:1.2.3.4 ip4:5.6.7.8 -all",
		},
		{
			description: "include ending in -all drops the all",
			dnsres:      fakeResolver{"example.net": "v=spf1 ip4:1.2.3.4 -all"},
			input:       "v=spf1 include:example.net -all",
			want:        "v=spf1 ip4:1.2.3.4 -all",
		},
		{
			description: "include ending in ?all drops the all",
			dnsres:      fakeResolver{"example.net": "v=spf1 ip4:1.2.3.4 ?all"},
			input:       "v=spf1 include:example.net -all",
			want:        "v=spf1 ip4:1.2.3.4 -all",
		},
		{
			description: "include ending in unqualified all drops the all",
			dnsres:      fakeResolver{"example.net": "v=spf1 ip4:1.2.3.4 all"},
			input:       "v=spf1 include:example.net -all",
			want:        "v=spf1 ip4:1.2.3.4 -all",
		},
		{
			description: "include ending in -ALL drops the all",
			dnsres:      fakeResolver{"example.net": "v=spf1 ip4:1.2.3.4 -ALL"},
			input:       "v=spf1 include:example.net -all",
			want:        "v=spf1 ip4:1.2.3.4 -all",
		},
		{
			description: "nested include with no all keeps its last term",
			dnsres: fakeResolver{
				"a.example.net": "v=spf1 include:b.example.net ip4:1.2.3.4",
				"b.example.net": "v=spf1 ip4:5.6.7.8 -all",
			},
			input: "v=spf1 include:a.example.net -all",
			want:  "v=spf1 ip4:5.6.7.8 ip4:1.2.3.4 -all",
		},
		{
			description: "include of a target that only includes a no-mail domain",
			dnsres: fakeResolver{
				"a.example.net": "v=spf1 include:b.example.net",
				"b.example.net": "v=spf1 -all",
			},
			input: "v=spf1 include:a.example.net -all",
			want:  "v=spf1 -all",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			rec, err := Parse(test.input, test.dnsres)
			if err != nil {
				t.Fatal(err)
			}
			if got := rec.Flatten("*").TXT(); got != test.want {
				t.Errorf("got %s want %s", got, test.want)
			}
		})
	}
}

// each test is array of strings.
// first item is unsplit input
// next is @ spf record
// after that is alternating record fqdn and value.
var splitTests = [][]string{
	{
		"simple",
		"v=spf1 -all",
		"v=spf1 -all",
	},
	{
		"longsimple",
		"v=spf1 include:a01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
		"v=spf1 include:a01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
	},
	{
		"long simple multipart",
		"v=spf1 include:a.com include:b.com include:12345678901234567890123456789000000000000000123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
		"v=spf1 include:a.com include:b.com include:12345678901234567890123456789000000000000000123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
	},
	{
		"overflow",
		"v=spf1 include:a.com include:b.com include:X12345678901234567890123456789000000000000000123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
		"v=spf1 include:a.com include:b.com include:_spf1.stackex.com -all",
		"_spf1.stackex.com",
		"v=spf1 include:X12345678901234567890123456789000000000000000123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
	},
	{
		"overflow all sign carries",
		"v=spf1 include:a.com include:b.com include:X12345678901234567890123456789000000000000000123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com ~all",
		"v=spf1 include:a.com include:b.com include:_spf1.stackex.com ~all",
		"_spf1.stackex.com",
		"v=spf1 include:X12345678901234567890123456789000000000000000123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com ~all",
	},
	{
		"overhead-200",
		"v=spf1 include:a.com include:b.com include:X12345678901234567890123456789000000000000000123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com ~all",
		"v=spf1 include:a.com include:_spf1.stackex.com ~all",
		"_spf1.stackex.com",
		"v=spf1 include:b.com include:X12345678901234567890123456789000000000000000123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com ~all",
	},
	{
		"really big",
		"v=spf1 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178" +
			" ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178" +
			" ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 -all",
		"v=spf1 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 include:_spf1.stackex.com -all",
		"_spf1.stackex.com",
		"v=spf1 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 include:_spf2.stackex.com -all",
		"_spf2.stackex.com",
		"v=spf1 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 include:_spf3.stackex.com -all",
		"_spf3.stackex.com",
		"v=spf1 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 -all",
	},
	{
		"too long to split",
		"v=spf1 include:a0123456789012345678901234567890123456789012345sssss6789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
		"v=spf1 include:a0123456789012345678901234567890123456789012345sssss6789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
	},
}

func TestSplit(t *testing.T) {
	for _, tst := range splitTests {
		overhead := 0
		txtMaxSize := 255

		v := strings.Split(tst[0], "-")
		if len(v) > 1 {
			if i, err := strconv.Atoi(v[1]); err == nil {
				overhead = i
			}
		}

		t.Run(tst[0], func(t *testing.T) {
			rec, err := Parse(tst[1], nil)
			if err != nil {
				t.Fatal(err)
			}
			res := rec.TXTSplit("_spf%d.stackex.com", overhead, txtMaxSize)
			if res["@"][0] != tst[2] {
				t.Fatalf("Root record wrong. \nExp %s\ngot %s", tst[2], res["@"][0])
			}
			for i := 3; i < len(tst); i += 2 {
				fqdn := tst[i]
				exp := tst[i+1]
				if res[fqdn][0] != exp {
					t.Fatalf("Record %s.\nExp %s\ngot %s", fqdn, exp, res[fqdn])
				}
			}
		})
	}
}

func TestMultiStringSplit(t *testing.T) {
	var tests = []struct {
		description  string
		unsplitInput string
		txtMaxSize   int
		want         map[string][]string
	}{
		{
			description:  "basic multiple strings",
			unsplitInput: "v=spf1 -all",
			txtMaxSize:   255,
			want: map[string][]string{
				"@": {
					"v=spf1 -all",
				},
			},
		},
		{
			description:  "too long to split, not enough txtMaxSize for a multi string",
			unsplitInput: "v=spf1 include:a0123456789012345678901234567890123456789012345sssss6789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
			txtMaxSize:   255,
			want: map[string][]string{
				"@": {
					"v=spf1 include:a0123456789012345678901234567890123456789012345sssss6789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
				},
			},
		},
		{
			description:  "can actually be split with multiple txt strings",
			unsplitInput: "v=spf1 include:a0123456789012345678901234567890123456789012345sssss6789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com -all",
			txtMaxSize:   450,
			want: map[string][]string{
				"@": {
					// First string
					"v=spf1 include:a0123456789012345678901234567890123456789012345sssss6789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.com",
					// Second string
					" -all",
				},
			},
		},
		{
			description: "really big with multiple txt strings",
			unsplitInput: "v=spf1 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178" +
				" ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178" +
				" ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 -all",
			txtMaxSize: 450,
			want: map[string][]string{
				"@": {
					"v=spf1 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.",
					"192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 include:_spf1.stackex.com -all",
				},
				"_spf1.stackex.com": {
					"v=spf1 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.",
					"192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 ip4:200.192.169.178 -all",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			rec, err := Parse(test.unsplitInput, nil)
			if err != nil {
				t.Fatal(err)
			}
			got := rec.TXTSplit("_spf%d.stackex.com", 0, test.txtMaxSize)

			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %v want %v", got, test.want)
			}
		})
	}
}
