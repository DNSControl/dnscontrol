package models

import (
	"maps"
	"strings"
	"testing"
)

// The options object arrives from dnsconfig.js as JSON, so the test data uses
// the types json.Unmarshal produces: string, bool, float64 and map[string]any.

func TestBuilderMigadu(t *testing.T) {
	tests := []struct {
		name          string
		zone          string
		subdomain     string
		wantSubDomain string
		ttl           uint32
		args          []any
		want          []string
	}{
		{
			name: "apex, defaults",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com"},
			want: []string{
				"@ 300 IN MX 10 aspmx1.migadu.com.",
				"@ 300 IN MX 20 aspmx2.migadu.com.",
				"key1._domainkey 300 IN CNAME key1.example.com._domainkey.migadu.com.",
				"key2._domainkey 300 IN CNAME key2.example.com._domainkey.migadu.com.",
				"key3._domainkey 300 IN CNAME key3.example.com._domainkey.migadu.com.",
				"autoconfig 300 IN CNAME autoconfig.migadu.com.",
				"_autodiscover._tcp 300 IN SRV 0 1 443 autodiscover.migadu.com.",
				"_submissions._tcp 300 IN SRV 0 1 465 smtp.migadu.com.",
				"_imaps._tcp 300 IN SRV 0 1 993 imap.migadu.com.",
				"_pop3s._tcp 300 IN SRV 0 1 995 pop.migadu.com.",
			},
		},
		{
			name: "apex, verification and SPF",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"verify": "wxyz5678",
				"spf":    true,
			}},
			want: []string{
				`@ 300 IN TXT "hosted-email-verify=wxyz5678"`,
				"@ 300 IN MX 10 aspmx1.migadu.com.",
				"@ 300 IN MX 20 aspmx2.migadu.com.",
				"key1._domainkey 300 IN CNAME key1.example.com._domainkey.migadu.com.",
				"key2._domainkey 300 IN CNAME key2.example.com._domainkey.migadu.com.",
				"key3._domainkey 300 IN CNAME key3.example.com._domainkey.migadu.com.",
				`@ 300 IN TXT "v=spf1 include:spf.migadu.com -all"`,
				"autoconfig 300 IN CNAME autoconfig.migadu.com.",
				"_autodiscover._tcp 300 IN SRV 0 1 443 autodiscover.migadu.com.",
				"_submissions._tcp 300 IN SRV 0 1 465 smtp.migadu.com.",
				"_imaps._tcp 300 IN SRV 0 1 993 imap.migadu.com.",
				"_pop3s._tcp 300 IN SRV 0 1 995 pop.migadu.com.",
			},
		},
		{
			// The DKIM targets embed the mail domain, not the zone.
			name: "below a label, wildcard MX",
			zone: "example.com",
			ttl:  3600,
			args: []any{"mail.example.com", map[string]any{
				"label":        "mail",
				"verify":       "abcd1234",
				"wildcard":     true,
				"autoconfig":   false,
				"autodiscover": false,
				"clientSRV":    false,
			}},
			want: []string{
				`mail 3600 IN TXT "hosted-email-verify=abcd1234"`,
				"mail 3600 IN MX 10 aspmx1.migadu.com.",
				"mail 3600 IN MX 20 aspmx2.migadu.com.",
				"*.mail 3600 IN MX 10 aspmx1.migadu.com.",
				"*.mail 3600 IN MX 20 aspmx2.migadu.com.",
				"key1._domainkey.mail 3600 IN CNAME key1.mail.example.com._domainkey.migadu.com.",
				"key2._domainkey.mail 3600 IN CNAME key2.mail.example.com._domainkey.migadu.com.",
				"key3._domainkey.mail 3600 IN CNAME key3.mail.example.com._domainkey.migadu.com.",
			},
		},
		{
			name: "apex wildcard",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"wildcard":     true,
				"dkim":         false,
				"autoconfig":   false,
				"autodiscover": false,
				"clientSRV":    false,
			}},
			want: []string{
				"@ 300 IN MX 10 aspmx1.migadu.com.",
				"@ 300 IN MX 20 aspmx2.migadu.com.",
				"* 300 IN MX 10 aspmx1.migadu.com.",
				"* 300 IN MX 20 aspmx2.migadu.com.",
			},
		},
		{
			name:          "D_EXTEND",
			zone:          "example.com",
			subdomain:     "sub",
			wantSubDomain: "sub",
			ttl:           300,
			args: []any{"sub.example.com", map[string]any{
				"autoconfig":   false,
				"autodiscover": false,
				"clientSRV":    false,
			}},
			want: []string{
				"sub 300 IN MX 10 aspmx1.migadu.com.",
				"sub 300 IN MX 20 aspmx2.migadu.com.",
				"key1._domainkey.sub 300 IN CNAME key1.sub.example.com._domainkey.migadu.com.",
				"key2._domainkey.sub 300 IN CNAME key2.sub.example.com._domainkey.migadu.com.",
				"key3._domainkey.sub 300 IN CNAME key3.sub.example.com._domainkey.migadu.com.",
			},
		},
		{
			name:          "D_EXTEND, written in upper case",
			zone:          "example.com",
			subdomain:     "SUB",
			wantSubDomain: "sub",
			ttl:           300,
			args: []any{"SUB.Example.com", map[string]any{
				"mx":           false,
				"autoconfig":   false,
				"autodiscover": false,
				"clientSRV":    false,
			}},
			want: []string{
				"key1._domainkey.sub 300 IN CNAME key1.sub.example.com._domainkey.migadu.com.",
				"key2._domainkey.sub 300 IN CNAME key2.sub.example.com._domainkey.migadu.com.",
				"key3._domainkey.sub 300 IN CNAME key3.sub.example.com._domainkey.migadu.com.",
			},
		},
		{
			// The DKIM targets carry the punycode form, the only form DNS has.
			name: "internationalized domain name",
			zone: "xn--mnchen-3ya.de",
			ttl:  300,
			args: []any{"münchen.de", map[string]any{
				"mx":           false,
				"autoconfig":   false,
				"autodiscover": false,
				"clientSRV":    false,
			}},
			want: []string{
				"key1._domainkey 300 IN CNAME key1.xn--mnchen-3ya.de._domainkey.migadu.com.",
				"key2._domainkey 300 IN CNAME key2.xn--mnchen-3ya.de._domainkey.migadu.com.",
				"key3._domainkey 300 IN CNAME key3.xn--mnchen-3ya.de._domainkey.migadu.com.",
			},
		},
		{
			name: "every record off",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"mx":           false,
				"dkim":         false,
				"autoconfig":   false,
				"autodiscover": false,
				"clientSRV":    false,
			}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := MustNewDomainConfig(tt.zone)

			records, err := BuilderMigadu(dc, tt.ttl, tt.args, tt.subdomain)
			if err != nil {
				t.Fatalf("BuilderMigadu() error = %v, want none", err)
			}

			var got []string
			for _, rec := range records {
				got = append(got, rec.LineString())
			}
			if len(got) != len(tt.want) {
				t.Fatalf("BuilderMigadu() created %d records, want %d:\ngot:  %q\nwant: %q", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("record %d = %q, want %q", i, got[i], tt.want[i])
				}
				if records[i].SubDomain != tt.wantSubDomain {
					t.Errorf("record %d SubDomain = %q, want %q", i, records[i].SubDomain, tt.wantSubDomain)
				}
			}
		})
	}
}

func TestBuilderMigaduErrors(t *testing.T) {
	tests := []struct {
		name      string
		zone      string
		subdomain string
		args      []any
		wantErr   string
	}{
		{
			name:    "no arguments",
			zone:    "example.com",
			args:    []any{},
			wantErr: `MIGADU_BUILDER: the first argument must be the mail domain name, for example MIGADU_BUILDER("example.com", { verify: "abcd1234" })`,
		},
		{
			name:    "first argument is not a string",
			zone:    "example.com",
			args:    []any{map[string]any{"verify": "abcd1234"}},
			wantErr: `MIGADU_BUILDER: the first argument must be the mail domain name, for example MIGADU_BUILDER("example.com", { verify: "abcd1234" })`,
		},
		{
			name:    "too many arguments",
			zone:    "example.com",
			args:    []any{"example.com", map[string]any{}, map[string]any{}},
			wantErr: `MIGADU_BUILDER("example.com"): expected the domain name and one options object, got 3 arguments`,
		},
		{
			name:    "second argument is not an object",
			zone:    "example.com",
			args:    []any{"example.com", "abcd1234"},
			wantErr: `MIGADU_BUILDER("example.com"): the second argument must be an options object, got string`,
		},
		{
			// There is no ttl option; TTL() on the call sets the TTL.
			name:    "unknown option",
			zone:    "example.com",
			args:    []any{"example.com", map[string]any{"ttl": "1h"}},
			wantErr: `MIGADU_BUILDER("example.com"): unknown option "ttl"; valid options are autoconfig, autodiscover, clientSRV, dkim, label, mx, spf, verify, wildcard`,
		},
		{
			name:    "boolean option is a string",
			zone:    "example.com",
			args:    []any{"example.com", map[string]any{"spf": "true"}},
			wantErr: `MIGADU_BUILDER("example.com"): option "spf" must be true or false`,
		},
		{
			name:    "empty label",
			zone:    "example.com",
			args:    []any{"example.com", map[string]any{"label": " "}},
			wantErr: `MIGADU_BUILDER("example.com"): option "label" must not be empty`,
		},
		{
			// The admin panel shows the whole TXT value; pasting it would
			// otherwise publish the prefix twice.
			name:    "verify is the whole TXT value",
			zone:    "example.com",
			args:    []any{"example.com", map[string]any{"verify": "hosted-email-verify=wxyz5678"}},
			wantErr: `MIGADU_BUILDER("example.com"): option "verify" must be the token only, without the "hosted-email-verify=" prefix, for example "abcd1234"`,
		},
		{
			name:    "verify is a number",
			zone:    "example.com",
			args:    []any{"example.com", map[string]any{"verify": float64(1234)}},
			wantErr: `MIGADU_BUILDER("example.com"): option "verify" must be a string`,
		},
		{
			name:    "wildcard without MX",
			zone:    "example.com",
			args:    []any{"example.com", map[string]any{"wildcard": true, "mx": false}},
			wantErr: `MIGADU_BUILDER("example.com"): option "wildcard" adds MX records, so it cannot be combined with "mx": false`,
		},
		{
			// The first argument is the mail domain, not the zone.
			name:    "label without the full mail domain",
			zone:    "example.com",
			args:    []any{"example.com", map[string]any{"label": "mail"}},
			wantErr: `MIGADU_BUILDER("example.com"): the mail domain "example.com" is not the name these records are created under ("mail.example.com"); pass that name as the first argument, or set "label"`,
		},
		{
			name:    "name does not match the zone",
			zone:    "example.com",
			args:    []any{"other.com"},
			wantErr: `MIGADU_BUILDER("other.com"): the mail domain "other.com" is not the name these records are created under ("example.com"); pass that name as the first argument, or set "label"`,
		},
		{
			name:      "D_EXTEND with the parent zone's name",
			zone:      "example.com",
			subdomain: "sub",
			args:      []any{"example.com"},
			wantErr:   `MIGADU_BUILDER("example.com"): the mail domain "example.com" is not the name these records are created under ("sub.example.com"); pass that name as the first argument, or set "label"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := MustNewDomainConfig(tt.zone)

			records, err := BuilderMigadu(dc, 300, tt.args, tt.subdomain)
			if err == nil {
				t.Fatalf("BuilderMigadu() created %d records, want error %q", len(records), tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("BuilderMigadu() error =\n%q\nwant:\n%q", err.Error(), tt.wantErr)
			}
		})
	}
}

// One options object may be handed to several D() blocks; the builder must
// not write anything back into it.
func TestBuilderMigaduSharedOptions(t *testing.T) {
	opts := map[string]any{"verify": "abcd1234"}
	before := maps.Clone(opts)

	for _, zone := range []string{"first.com", "second.com"} {
		records, err := BuilderMigadu(MustNewDomainConfig(zone), 300, []any{zone, opts}, "")
		if err != nil {
			t.Fatalf("BuilderMigadu(%q) error = %v, want none", zone, err)
		}
		want := "key1._domainkey 300 IN CNAME key1." + zone + "._domainkey.migadu.com."
		found := false
		for _, rec := range records {
			if rec.LineString() == want {
				found = true
			}
			if strings.Contains(rec.LineString(), "first.com") && zone != "first.com" {
				t.Errorf("%s: record %q leaks the first domain", zone, rec.LineString())
			}
		}
		if !found {
			t.Errorf("%s: no record %q", zone, want)
		}
	}

	if !maps.Equal(opts, before) {
		t.Errorf("options object = %v, want it unchanged: %v", opts, before)
	}
}
