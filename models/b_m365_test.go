package models

import (
	"maps"
	"strings"
	"testing"
)

// The options object arrives from dnsconfig.js as JSON, so the test data uses
// the types json.Unmarshal produces: string, bool, float64 and map[string]any.
// An int literal would test a case that cannot happen.

func TestBuilderM365(t *testing.T) {
	tests := []struct {
		name string
		zone string
		// subdomain is what D_EXTEND() hands to the builder; wantSubDomain is
		// what the records carry, which is the IDNA form of it.
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
			args: []any{"example.com", map[string]any{
				"initialDomain": "contoso.onmicrosoft.com",
			}},
			want: []string{
				"@ 300 IN MX 0 example-com.mail.protection.outlook.com.",
				"autodiscover 300 IN CNAME autodiscover.outlook.com.",
				"selector1._domainkey 300 IN CNAME selector1-example-com._domainkey.contoso.onmicrosoft.com.",
				"selector2._domainkey 300 IN CNAME selector2-example-com._domainkey.contoso.onmicrosoft.com.",
			},
		},
		{
			name: "every record, below a label",
			zone: "example.com",
			ttl:  3600,
			args: []any{"test.example.com", map[string]any{
				"label":             "test",
				"initialDomain":     "contoso.onmicrosoft.com",
				"verificationToken": "MS=ms12345678",
				"skypeForBusiness":  true,
				"mdm":               true,
			}},
			want: []string{
				`test 3600 IN TXT "MS=ms12345678"`,
				"test 3600 IN MX 0 test-example-com.mail.protection.outlook.com.",
				"autodiscover.test 3600 IN CNAME autodiscover.outlook.com.",
				"selector1._domainkey.test 3600 IN CNAME selector1-test-example-com._domainkey.contoso.onmicrosoft.com.",
				"selector2._domainkey.test 3600 IN CNAME selector2-test-example-com._domainkey.contoso.onmicrosoft.com.",
				"_sipfederationtls._tcp.test 3600 IN SRV 100 1 5061 sipfed.online.lync.com.",
				"enterpriseregistration.test 3600 IN CNAME enterpriseregistration.windows.net.",
				"enterpriseenrollment.test 3600 IN CNAME enterpriseenrollment-s.manage.microsoft.com.",
			},
		},
		{
			name:          "D_EXTEND",
			zone:          "example.com",
			subdomain:     "sub",
			wantSubDomain: "sub",
			ttl:           300,
			args: []any{"sub.example.com", map[string]any{
				"initialDomain": "contoso.onmicrosoft.com",
			}},
			want: []string{
				"sub 300 IN MX 0 sub-example-com.mail.protection.outlook.com.",
				"autodiscover.sub 300 IN CNAME autodiscover.outlook.com.",
				"selector1._domainkey.sub 300 IN CNAME selector1-sub-example-com._domainkey.contoso.onmicrosoft.com.",
				"selector2._domainkey.sub 300 IN CNAME selector2-sub-example-com._domainkey.contoso.onmicrosoft.com.",
			},
		},
		{
			// Upper case is meaningless in DNS, so neither the domain name nor
			// the D_EXTEND() subdomain may depend on how it was written.
			name:          "D_EXTEND, written in upper case",
			zone:          "example.com",
			subdomain:     "SUB",
			wantSubDomain: "sub",
			ttl:           300,
			args: []any{"SUB.example.com", map[string]any{
				"initialDomain": "contoso.onmicrosoft.com",
			}},
			want: []string{
				"sub 300 IN MX 0 sub-example-com.mail.protection.outlook.com.",
				"autodiscover.sub 300 IN CNAME autodiscover.outlook.com.",
				"selector1._domainkey.sub 300 IN CNAME selector1-sub-example-com._domainkey.contoso.onmicrosoft.com.",
				"selector2._domainkey.sub 300 IN CNAME selector2-sub-example-com._domainkey.contoso.onmicrosoft.com.",
			},
		},
		{
			name: "DKIM targets assigned by Microsoft",
			zone: "contoso.com",
			ttl:  300,
			args: []any{"contoso.com", map[string]any{
				"dkimSelector1Target": "selector1-contoso-com._domainkey.contoso.n-v1.dkim.mail.microsoft",
				"dkimSelector2Target": "selector2-contoso-com._domainkey.contoso.n-v1.dkim.mail.microsoft",
			}},
			want: []string{
				"@ 300 IN MX 0 contoso-com.mail.protection.outlook.com.",
				"autodiscover 300 IN CNAME autodiscover.outlook.com.",
				"selector1._domainkey 300 IN CNAME selector1-contoso-com._domainkey.contoso.n-v1.dkim.mail.microsoft.",
				"selector2._domainkey 300 IN CNAME selector2-contoso-com._domainkey.contoso.n-v1.dkim.mail.microsoft.",
			},
		},
		{
			// The domain token is needed here even though no MX record is
			// created, because the DKIM targets are derived from it.
			name: "DKIM without MX",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"mx":            false,
				"autodiscover":  false,
				"initialDomain": "contoso.onmicrosoft.com",
			}},
			want: []string{
				"selector1._domainkey 300 IN CNAME selector1-example-com._domainkey.contoso.onmicrosoft.com.",
				"selector2._domainkey 300 IN CNAME selector2-example-com._domainkey.contoso.onmicrosoft.com.",
			},
		},
		{
			name: "DANE and DNSSEC",
			zone: "contosotest.com",
			ttl:  300,
			args: []any{"contosotest.com", map[string]any{
				"mxTarget":            "contosotest-com.o-v1.mx.microsoft",
				"dkimSelector1Target": "selector1-contosotest-com._domainkey.contoso.o-v1.dkim.mail.microsoft",
				"dkimSelector2Target": "selector2-contosotest-com._domainkey.contoso.o-v1.dkim.mail.microsoft",
			}},
			want: []string{
				"@ 300 IN MX 0 contosotest-com.o-v1.mx.microsoft.",
				"autodiscover 300 IN CNAME autodiscover.outlook.com.",
				"selector1._domainkey 300 IN CNAME selector1-contosotest-com._domainkey.contoso.o-v1.dkim.mail.microsoft.",
				"selector2._domainkey 300 IN CNAME selector2-contosotest-com._domainkey.contoso.o-v1.dkim.mail.microsoft.",
			},
		},
		{
			name: "only MDM, domain name with a dash",
			zone: "my-example.com",
			ttl:  300,
			args: []any{"my-example.com", map[string]any{
				"mx":           false,
				"autodiscover": false,
				"dkim":         false,
				"mdm":          true,
			}},
			want: []string{
				"enterpriseregistration 300 IN CNAME enterpriseregistration.windows.net.",
				"enterpriseenrollment 300 IN CNAME enterpriseenrollment-s.manage.microsoft.com.",
			},
		},
		{
			name: "both device management targets set explicitly",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"mx":                    false,
				"autodiscover":          false,
				"dkim":                  false,
				"mdm":                   true,
				"mdmEnrollmentTarget":   "enterpriseenrollment-s.manage.microsoft.us",
				"mdmRegistrationTarget": "enterpriseregistration.windows.net",
			}},
			want: []string{
				"enterpriseregistration 300 IN CNAME enterpriseregistration.windows.net.",
				"enterpriseenrollment 300 IN CNAME enterpriseenrollment-s.manage.microsoft.us.",
			},
		},
		{
			name: "GCC High",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"mxTarget":            "contoso.mail.protection.office365.us",
				"autodiscoverTarget":  "autodiscover.office365.us",
				"dkim":                false,
				"skypeForBusiness":    true,
				"sipFederationTarget": "sipfed.online.gov.skypeforbusiness.us",
				"mdm":                 true,
				"mdmEnrollmentTarget": "enterpriseenrollment-s.manage.microsoft.us",
			}},
			want: []string{
				"@ 300 IN MX 0 contoso.mail.protection.office365.us.",
				"autodiscover 300 IN CNAME autodiscover.office365.us.",
				"_sipfederationtls._tcp 300 IN SRV 100 1 5061 sipfed.online.gov.skypeforbusiness.us.",
				"enterpriseregistration 300 IN CNAME enterpriseregistration.windows.net.",
				"enterpriseenrollment 300 IN CNAME enterpriseenrollment-s.manage.microsoft.us.",
			},
		},
		{
			name: "every record switched off",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"mx":           false,
				"autodiscover": false,
				"dkim":         false,
			}},
			want: nil,
		},
		{
			// A selector target left behind after DKIM was switched off
			// describes no record, so it must not raise the pairing error.
			name: "one DKIM target left behind while DKIM is off",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"mx":                  false,
				"autodiscover":        false,
				"dkim":                false,
				"dkimSelector1Target": "selector1-example-com._domainkey.contoso.n-v1.dkim.mail.microsoft",
			}},
			want: nil,
		},
		{
			// Values copied out of the admin center arrive with stray spaces
			// and with a trailing dot that the user may have typed twice.
			name: "surrounding space and trailing dots are accepted everywhere",
			zone: "example.com",
			ttl:  300,
			args: []any{"  example.com..  ", map[string]any{
				"initialDomain":      "  contoso.onmicrosoft.com..  ",
				"autodiscoverTarget": "autodiscover.office365.us..",
			}},
			want: []string{
				"@ 300 IN MX 0 example-com.mail.protection.outlook.com.",
				"autodiscover 300 IN CNAME autodiscover.office365.us.",
				"selector1._domainkey 300 IN CNAME selector1-example-com._domainkey.contoso.onmicrosoft.com.",
				"selector2._domainkey 300 IN CNAME selector2-example-com._domainkey.contoso.onmicrosoft.com.",
			},
		},
		{
			name: "domainGUID and mxPriority from the admin center",
			zone: "my-example.com",
			ttl:  300,
			args: []any{"my-example.com", map[string]any{
				"domainGUID":    "myexample-com01c",
				"mxPriority":    float64(20),
				"initialDomain": "contoso.onmicrosoft.com",
			}},
			want: []string{
				"@ 300 IN MX 20 myexample-com01c.mail.protection.outlook.com.",
				"autodiscover 300 IN CNAME autodiscover.outlook.com.",
				"selector1._domainkey 300 IN CNAME selector1-myexample-com01c._domainkey.contoso.onmicrosoft.com.",
				"selector2._domainkey 300 IN CNAME selector2-myexample-com01c._domainkey.contoso.onmicrosoft.com.",
			},
		},
		{
			// label and the D_EXTEND() subdomain stack: the label is relative
			// to the extended name, and the domain token spans both.
			name:          "a label below a D_EXTEND() subdomain",
			zone:          "example.com",
			subdomain:     "sub",
			wantSubDomain: "sub",
			ttl:           300,
			args: []any{"test.sub.example.com", map[string]any{
				"label":         "test",
				"initialDomain": "contoso.onmicrosoft.com",
			}},
			want: []string{
				"test.sub 300 IN MX 0 test-sub-example-com.mail.protection.outlook.com.",
				"autodiscover.test.sub 300 IN CNAME autodiscover.outlook.com.",
				"selector1._domainkey.test.sub 300 IN CNAME selector1-test-sub-example-com._domainkey.contoso.onmicrosoft.com.",
				"selector2._domainkey.test.sub 300 IN CNAME selector2-test-sub-example-com._domainkey.contoso.onmicrosoft.com.",
			},
		},
		{
			// A label inside the zone may be written as a fully qualified name.
			name: "label written as a fully qualified name",
			zone: "example.com",
			ttl:  300,
			args: []any{"test.example.com", map[string]any{
				"label":         "test.example.com.",
				"initialDomain": "contoso.onmicrosoft.com",
			}},
			want: []string{
				"test 300 IN MX 0 test-example-com.mail.protection.outlook.com.",
				"autodiscover.test 300 IN CNAME autodiscover.outlook.com.",
				"selector1._domainkey.test 300 IN CNAME selector1-test-example-com._domainkey.contoso.onmicrosoft.com.",
				"selector2._domainkey.test 300 IN CNAME selector2-test-example-com._domainkey.contoso.onmicrosoft.com.",
			},
		},
		{
			name: "DoD",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"mxTarget":            "contoso.mail.protection.office365.us",
				"autodiscoverTarget":  "autodiscover-dod.office365.us",
				"dkim":                false,
				"skypeForBusiness":    true,
				"sipFederationTarget": "sipfed.online.dod.skypeforbusiness.us",
			}},
			want: []string{
				"@ 300 IN MX 0 contoso.mail.protection.office365.us.",
				"autodiscover 300 IN CNAME autodiscover-dod.office365.us.",
				"_sipfederationtls._tcp 300 IN SRV 100 1 5061 sipfed.online.dod.skypeforbusiness.us.",
			},
		},
		{
			name: "21Vianet device management",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"mx":                  false,
				"autodiscover":        false,
				"dkim":                false,
				"mdm":                 true,
				"mdmEnrollmentTarget": "enterpriseenrollment-s.manage.microsoftonline.cn",
			}},
			want: []string{
				"enterpriseregistration 300 IN CNAME enterpriseregistration.windows.net.",
				"enterpriseenrollment 300 IN CNAME enterpriseenrollment-s.manage.microsoftonline.cn.",
			},
		},
		{
			// The largest preference the record can carry.
			name: "mxPriority at the top of its range",
			zone: "example.com",
			ttl:  300,
			args: []any{"example.com", map[string]any{
				"autodiscover": false,
				"dkim":         false,
				"mxPriority":   float64(65535),
			}},
			want: []string{
				"@ 300 IN MX 65535 example-com.mail.protection.outlook.com.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := MustNewDomainConfig(tt.zone)

			records, err := BuilderM365(dc, tt.ttl, tt.args, tt.subdomain)
			if err != nil {
				t.Fatalf("BuilderM365() error = %v, want none", err)
			}

			var got []string
			for _, rec := range records {
				got = append(got, rec.LineString())
			}
			if len(got) != len(tt.want) {
				t.Fatalf("BuilderM365() created %d records, want %d:\ngot:  %q\nwant: %q", len(got), len(tt.want), got, tt.want)
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

func TestBuilderM365Errors(t *testing.T) {
	tests := []struct {
		name      string
		zone      string
		subdomain string
		args      []any
		want      string
		// wantPrefix replaces want where the message ends in the wording of
		// another library, which is not ours to pin down.
		wantPrefix string
	}{
		{
			name: "E1 no arguments",
			zone: "example.com",
			args: []any{},
			want: `M365_BUILDER: the first argument must be the Microsoft 365 domain name, for example M365_BUILDER("example.com", { initialDomain: "contoso.onmicrosoft.com" })`,
		},
		{
			name: "E1 first argument is all whitespace",
			zone: "example.com",
			args: []any{"   "},
			want: `M365_BUILDER: the first argument must be the Microsoft 365 domain name, for example M365_BUILDER("example.com", { initialDomain: "contoso.onmicrosoft.com" })`,
		},
		{
			name: "E1 first argument is not a string",
			zone: "example.com",
			args: []any{map[string]any{"initialDomain": "contoso.onmicrosoft.com"}},
			want: `M365_BUILDER: the first argument must be the Microsoft 365 domain name, for example M365_BUILDER("example.com", { initialDomain: "contoso.onmicrosoft.com" })`,
		},
		{
			name: "E2a too many arguments",
			zone: "example.com",
			args: []any{"example.com", map[string]any{}, map[string]any{}},
			want: `M365_BUILDER("example.com"): expected the domain name and one options object, got 3 arguments`,
		},
		{
			name: "E2b second argument is not an object",
			zone: "example.com",
			args: []any{"example.com", "contoso.onmicrosoft.com"},
			want: `M365_BUILDER("example.com"): the second argument must be an options object, got string`,
		},
		{
			name: "E2b second argument is a number",
			zone: "example.com",
			args: []any{"example.com", float64(5)},
			want: `M365_BUILDER("example.com"): the second argument must be an options object, got number`,
		},
		{
			name: "E2b second argument is a boolean",
			zone: "example.com",
			args: []any{"example.com", true},
			want: `M365_BUILDER("example.com"): the second argument must be an options object, got boolean`,
		},
		{
			name: "E2b second argument is null",
			zone: "example.com",
			args: []any{"example.com", nil},
			want: `M365_BUILDER("example.com"): the second argument must be an options object, got null`,
		},
		{
			name: "E3 unknown option",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"initialdomain": "contoso.onmicrosoft.com"}},
			want: `M365_BUILDER("example.com"): unknown option "initialdomain"; valid options are autodiscover, autodiscoverTarget, dkim, dkimSelector1Target, dkimSelector2Target, domainGUID, initialDomain, label, mdm, mdmEnrollmentTarget, mdmRegistrationTarget, mx, mxPriority, mxTarget, sipFederationTarget, skypeForBusiness, verificationToken`,
		},
		{
			// Go maps have no order, so the options are validated sorted by
			// name. Both of these calls have to name the same option every
			// time, not a different one per run.
			name: "an unknown option is reported before a bad value",
			zone: "example.com",
			args: []any{"example.com", map[string]any{
				"zzUnknown":    true,
				"autodiscover": "yes",
			}},
			want: `M365_BUILDER("example.com"): unknown option "zzUnknown"; valid options are autodiscover, autodiscoverTarget, dkim, dkimSelector1Target, dkimSelector2Target, domainGUID, initialDomain, label, mdm, mdmEnrollmentTarget, mdmRegistrationTarget, mx, mxPriority, mxTarget, sipFederationTarget, skypeForBusiness, verificationToken`,
		},
		{
			name: "the alphabetically first bad option is reported",
			zone: "example.com",
			args: []any{"example.com", map[string]any{
				"mxPriority": "10",
				"dkim":       "false",
			}},
			want: `M365_BUILDER("example.com"): option "dkim" must be true or false`,
		},
		{
			name: "E4 switch is not a boolean",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"dkim": "false"}},
			want: `M365_BUILDER("example.com"): option "dkim" must be true or false`,
		},
		{
			name: "E5 string option is not a string",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"label": true}},
			want: `M365_BUILDER("example.com"): option "label" must be a string`,
		},
		{
			name: "E6 mxPriority is not a number",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"mxPriority": "10"}},
			want: `M365_BUILDER("example.com"): option "mxPriority" must be a number`,
		},
		{
			name: "E7 string option is empty",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"initialDomain": "  "}},
			want: `M365_BUILDER("example.com"): option "initialDomain" must not be empty`,
		},
		{
			name: "E8 mxPriority is not a whole number",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"mxPriority": 0.5}},
			want: `M365_BUILDER("example.com"): option "mxPriority" must be a whole number between 0 and 65535`,
		},
		{
			name: "E8 mxPriority is out of range",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"mxPriority": float64(65536)}},
			want: `M365_BUILDER("example.com"): option "mxPriority" must be a whole number between 0 and 65535`,
		},
		{
			name: "E8 mxPriority is negative",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"mxPriority": float64(-1)}},
			want: `M365_BUILDER("example.com"): option "mxPriority" must be a whole number between 0 and 65535`,
		},
		{
			name: "E9 target is not fully qualified",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"mxTarget": "contoso"}},
			want: `M365_BUILDER("example.com"): option "mxTarget" must be a fully qualified host name, for example "contoso.mail.protection.office365.us"`,
		},
		{
			name: "E9 initialDomain is not fully qualified",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"initialDomain": "contoso"}},
			want: `M365_BUILDER("example.com"): option "initialDomain" must be a fully qualified host name, for example "contoso.onmicrosoft.com"`,
		},
		{
			name: "E9 the only dot of a target is a leading one",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"mxTarget": ".com"}},
			want: `M365_BUILDER("example.com"): option "mxTarget" must be a fully qualified host name, for example "contoso.mail.protection.office365.us"`,
		},
		{
			name: "E9 autodiscoverTarget is not fully qualified",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"autodiscoverTarget": "autodiscover"}},
			want: `M365_BUILDER("example.com"): option "autodiscoverTarget" must be a fully qualified host name, for example "autodiscover.office365.us"`,
		},
		{
			name: "E9 dkimSelector1Target is not fully qualified",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"dkimSelector1Target": "selector1"}},
			want: `M365_BUILDER("example.com"): option "dkimSelector1Target" must be a fully qualified host name, for example "selector1-contoso-com._domainkey.contoso.n-v1.dkim.mail.microsoft"`,
		},
		{
			name: "E9 dkimSelector2Target is not fully qualified",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"dkimSelector2Target": "selector2"}},
			want: `M365_BUILDER("example.com"): option "dkimSelector2Target" must be a fully qualified host name, for example "selector2-contoso-com._domainkey.contoso.n-v1.dkim.mail.microsoft"`,
		},
		{
			name: "E9 mdmEnrollmentTarget is not fully qualified",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"mdmEnrollmentTarget": "enterpriseenrollment"}},
			want: `M365_BUILDER("example.com"): option "mdmEnrollmentTarget" must be a fully qualified host name, for example "enterpriseenrollment-s.manage.microsoft.us"`,
		},
		{
			name: "E9 mdmRegistrationTarget is not fully qualified",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"mdmRegistrationTarget": "enterpriseregistration"}},
			want: `M365_BUILDER("example.com"): option "mdmRegistrationTarget" must be a fully qualified host name, for example "enterpriseregistration.windows.net"`,
		},
		{
			name: "E9 sipFederationTarget is not fully qualified",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"sipFederationTarget": "sipfed"}},
			want: `M365_BUILDER("example.com"): option "sipFederationTarget" must be a fully qualified host name, for example "sipfed.online.gov.skypeforbusiness.us"`,
		},
		{
			name: "E10 domainGUID contains a dot",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"domainGUID": "example.com"}},
			want: `M365_BUILDER("example.com"): option "domainGUID" must be a single label without dots, for example "example-com"`,
		},
		{
			name: "E11 verificationToken without the MS= prefix",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"verificationToken": "ms12345678"}},
			want: `M365_BUILDER("example.com"): option "verificationToken" must be the full TXT value shown in the Microsoft 365 admin center, for example "MS=ms12345678"`,
		},
		{
			name: "E11 verificationToken is the MS= prefix alone",
			zone: "example.com",
			args: []any{"example.com", map[string]any{"verificationToken": "MS="}},
			want: `M365_BUILDER("example.com"): option "verificationToken" must be the full TXT value shown in the Microsoft 365 admin center, for example "MS=ms12345678"`,
		},
		{
			name: "E12 only one DKIM target",
			zone: "example.com",
			args: []any{"example.com", map[string]any{
				"dkimSelector1Target": "selector1-example-com._domainkey.contoso.n-v1.dkim.mail.microsoft",
			}},
			want: `M365_BUILDER("example.com"): options "dkimSelector1Target" and "dkimSelector2Target" must be set together; Microsoft publishes a value for both selectors`,
		},
		{
			name: "E13 label is not in the zone",
			zone: "example.com",
			args: []any{"example.com", map[string]any{
				"label":         "test.other.com.",
				"initialDomain": "contoso.onmicrosoft.com",
			}},
			want: `M365_BUILDER("example.com"): label "test.other.com." is not in domain "example.com"`,
		},
		{
			name:       "the D_EXTEND() subdomain is not a valid IDNA name",
			zone:       "example.com",
			subdomain:  "xn--x",
			args:       []any{"xn--x.example.com", map[string]any{"initialDomain": "contoso.onmicrosoft.com"}},
			wantPrefix: `M365_BUILDER("xn--x.example.com"): the D_EXTEND() subdomain "xn--x" is not a valid IDNA name: `,
		},
		{
			name:       "E14 domain name is not a valid IDNA name",
			zone:       "example.com",
			args:       []any{"xn--x.com", map[string]any{"initialDomain": "contoso.onmicrosoft.com"}},
			wantPrefix: `M365_BUILDER("xn--x.com"): the domain name is not a valid IDNA name: `,
		},
		{
			name:      "E15 domain name is not where the records are created",
			zone:      "example.com",
			subdomain: "sub",
			args:      []any{"example.com", map[string]any{"initialDomain": "contoso.onmicrosoft.com"}},
			want:      `M365_BUILDER("example.com"): the Microsoft 365 domain "example.com" is not the name these records are created under ("sub.example.com"); pass that name as the first argument, set "label", or set "domainGUID"`,
		},
		{
			name: "E16 domain name contains a dash",
			zone: "my-example.com",
			args: []any{"my-example.com", map[string]any{"initialDomain": "contoso.onmicrosoft.com"}},
			want: `M365_BUILDER("my-example.com"): "domainGUID" has no default for a domain name that contains a dash, because Microsoft assigns an opaque value (for example "myexample-com01c"); copy the value from the MX record shown in the Microsoft 365 admin center`,
		},
		{
			// Every internationalized name reaches this error, because its
			// punycode form begins with "xn--".
			name: "E16 internationalized domain name",
			zone: "xn--mnchen-3ya.de",
			args: []any{"münchen.de", map[string]any{"initialDomain": "contoso.onmicrosoft.com"}},
			want: `M365_BUILDER("münchen.de"): "domainGUID" has no default for a domain name that contains a dash, because Microsoft assigns an opaque value (for example "myexample-com01c"); copy the value from the MX record shown in the Microsoft 365 admin center`,
		},
		{
			name: "E17 DKIM without initialDomain",
			zone: "example.com",
			args: []any{"example.com", map[string]any{}},
			want: `M365_BUILDER("example.com"): option "initialDomain" is required to derive the DKIM targets ("dkim" defaults to true); set it to your tenant's initial domain, for example "contoso.onmicrosoft.com", set "dkimSelector1Target" and "dkimSelector2Target" to the values shown in the Microsoft 365 admin center, or set "dkim": false`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := MustNewDomainConfig(tt.zone)

			records, err := BuilderM365(dc, 300, tt.args, tt.subdomain)
			if err == nil {
				t.Fatalf("BuilderM365() created %d records, want error %q", len(records), tt.want)
			}
			if tt.wantPrefix != "" {
				if !strings.HasPrefix(err.Error(), tt.wantPrefix) {
					t.Errorf("BuilderM365() error =\n%q\nwant prefix:\n%q", err.Error(), tt.wantPrefix)
				}
				return
			}
			if err.Error() != tt.want {
				t.Errorf("BuilderM365() error =\n%q\nwant:\n%q", err.Error(), tt.want)
			}
		})
	}
}

// A dnsconfig.js may hand the same options object to several D() blocks. The
// builder must not write its derived values back into it, because the next
// domain would then get the first domain's MX and DKIM targets.
func TestBuilderM365SharedOptions(t *testing.T) {
	opts := map[string]any{"initialDomain": "contoso.onmicrosoft.com"}
	before := maps.Clone(opts)

	for _, zone := range []string{"first.com", "second.com"} {
		records, err := BuilderM365(MustNewDomainConfig(zone), 300, []any{zone, opts}, "")
		if err != nil {
			t.Fatalf("BuilderM365(%q) error = %v, want none", zone, err)
		}

		token := strings.ReplaceAll(zone, ".", "-")
		want := []string{
			"@ 300 IN MX 0 " + token + ".mail.protection.outlook.com.",
			"autodiscover 300 IN CNAME autodiscover.outlook.com.",
			"selector1._domainkey 300 IN CNAME selector1-" + token + "._domainkey.contoso.onmicrosoft.com.",
			"selector2._domainkey 300 IN CNAME selector2-" + token + "._domainkey.contoso.onmicrosoft.com.",
		}
		for i, rec := range records {
			if i < len(want) && rec.LineString() != want[i] {
				t.Errorf("%s: record %d = %q, want %q", zone, i, rec.LineString(), want[i])
			}
		}
		if len(records) != len(want) {
			t.Errorf("%s: created %d records, want %d", zone, len(records), len(want))
		}
	}

	if !maps.Equal(opts, before) {
		t.Errorf("options object = %v, want it unchanged: %v", opts, before)
	}
}
