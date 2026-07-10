package commands

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestRenderCredsJSON(t *testing.T) {
	cases := []struct {
		name     string
		existing map[string]map[string]string
		entries  []InitCredsEntry
		golden   string
	}{
		{
			name:     "single_cloudflare",
			existing: nil,
			entries: []InitCredsEntry{
				{
					Name:     "cloudflare",
					TypeName: "CLOUDFLAREAPI",
					Fields: map[string]string{
						"apitoken":  "tok-abc",
						"accountid": "acc-123",
					},
				},
			},
			golden: "creds_single_cloudflare.json",
		},
		{
			name: "merge_preserves_existing",
			existing: map[string]map[string]string{
				"bind": {"TYPE": "BIND", "directory": "zones"},
			},
			entries: []InitCredsEntry{
				{
					Name:     "cloudflare",
					TypeName: "CLOUDFLAREAPI",
					Fields:   map[string]string{"apitoken": "tok-abc"},
				},
			},
			golden: "creds_merge.json",
		},
		{
			name:     "none_entry",
			existing: nil,
			entries: []InitCredsEntry{
				{Name: "none", TypeName: "NONE", Fields: map[string]string{}},
			},
			golden: "creds_none.json",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := renderCredsJSON(testCase.existing, testCase.entries)
			if err != nil {
				t.Fatalf("renderCredsJSON: %v", err)
			}
			assertGolden(t, filepath.Join("testdata", "init", testCase.golden), got)
		})
	}
}

func TestRenderDnsconfigJS(t *testing.T) {
	cases := []struct {
		name   string
		choice InitDnsconfigChoice
		golden string
	}{
		{
			name: "registrar_and_dns",
			choice: InitDnsconfigChoice{
				RegistrarVar:  "REG_CLOUDFLAREAPI",
				RegistrarName: "cloudflare",
				DNSVar:        "DNS_CLOUDFLAREAPI",
				DNSName:       "cloudflare",
				Domains:       []string{"example.com", "example.org"},
			},
			golden: "config_cloudflare.js",
		},
		{
			name: "none_registrar_bind_dns",
			choice: InitDnsconfigChoice{
				RegistrarVar:  "REG_NONE",
				RegistrarName: "none",
				DNSVar:        "DNS_BIND",
				DNSName:       "bind",
				Domains:       []string{"example.com"},
			},
			golden: "config_bind.js",
		},
		{
			name: "registrar_only",
			choice: InitDnsconfigChoice{
				RegistrarVar:  "REG_NONE",
				RegistrarName: "none",
				Domains:       []string{"example.com"},
			},
			golden: "config_registrar_only.js",
		},
		{
			name: "imported_records",
			choice: InitDnsconfigChoice{
				RegistrarVar:  "REG_CLOUDFLAREAPI",
				RegistrarName: "cloudflare",
				DNSVar:        "DNS_CLOUDFLAREAPI",
				DNSName:       "cloudflare",
				Domains:       []string{"example.com"},
				DomainRecords: map[string]DomainImport{
					"example.com": {
						DefaultTTL: 3600,
						Records: []string{
							`A("@", "192.0.2.1")`,
							`A("www", "192.0.2.1")`,
							`MX("@", 10, "mx.example.com.")`,
						},
					},
				},
			},
			golden: "config_imported.js",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := renderDnsconfigJS(testCase.choice)
			assertGolden(t, filepath.Join("testdata", "init", testCase.golden), got)
		})
	}
}

func TestJSVarName(t *testing.T) {
	cases := []struct {
		prefix, input, want string
	}{
		{"REG", "CLOUDFLAREAPI", "REG_CLOUDFLAREAPI"},
		{"DNS", "HETZNER_V2", "DNS_HETZNER_V2"},
		{"DNS", "gandi-v5", "DNS_GANDI_V5"},
	}
	for _, testCase := range cases {
		if got := jsVarName(testCase.prefix, testCase.input); got != testCase.want {
			t.Errorf("jsVarName(%q,%q)=%q want %q", testCase.prefix, testCase.input, got, testCase.want)
		}
	}
}

// assertGolden compares got against the file at path. When the -update
// flag is passed the file is rewritten instead.
func assertGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *updateGolden {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s",
			path, string(got), string(want))
	}
}
