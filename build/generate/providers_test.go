package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func init() {
	for _, name := range []string{"GENERATOR_TEST_Z", "GENERATOR_TEST_A"} {
		providers.Register[*providers.None](name, providers.Definition{
			FriendlyName: "A test provider", Aliases: []string{name + "_ALIAS"},
			DocsURL: "https://docs.example.test/provider/" + name,
		})
	}
}

func TestReadmeUsesCanonicalDefinitions(t *testing.T) {
	t.Chdir(t.TempDir())
	const before = "Introduction\n<!-- provider-table-start -->\nold table\n<!-- provider-table-end -->\nOther sections\n"
	if err := os.WriteFile("README.md", []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := generateReadmeProvidersTable(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.HasPrefix(got, "Introduction\n<!-- provider-table-start -->") || !strings.HasSuffix(got, "<!-- provider-table-end -->\nOther sections\n") {
		t.Fatal("generation changed content outside the managed table")
	}
	if strings.Contains(got, "_ALIAS") || strings.Contains(got, "No registrar") {
		t.Fatal("aliases or the NONE placeholder appeared in the public provider table")
	}
	if strings.Count(got, "[A test provider]") != 2 {
		t.Fatal("canonical definitions were missing or counted more than once")
	}
	wantFirst := "| [A test provider](https://docs.example.test/provider/GENERATOR_TEST_A)² | [A test provider](https://docs.example.test/provider/GENERATOR_TEST_Z)² |"
	if !strings.Contains(got, wantFirst) {
		t.Fatal("friendly-name sorting or canonical-name tiebreaker was lost")
	}
	if !strings.Contains(got, fmt.Sprintf("supports %d DNS providers and registrars:", len(publicDefinitions()))) {
		t.Fatal("provider count does not match the canonical table")
	}
	for kind, suffix := range map[providers.ProviderKind]string{
		providers.KindDNS: "", providers.KindRegistrar: "²", providers.KindDNS | providers.KindRegistrar: "¹",
	} {
		def := &providers.Definition{TypeName: "NEW_TYPE", FriendlyName: "Brand", DocsURL: "https://example.com/old-path", Kind: kind}
		if got := readmeTableCell(def); got != "[Brand](https://example.com/old-path)"+suffix {
			t.Fatalf("table cell = %q", got)
		}
	}
}

func TestGeneratorPathsSurviveTypeRenames(t *testing.T) {
	for name, directory := range map[string]string{"CLOUDFLAREAPI": "cloudflare", "DNSOVERHTTPS": "doh", "HETZNER_V2": "hetznerv2"} {
		def, _ := providers.GetDefinition(name)
		renamed := *def
		renamed.TypeName, renamed.Aliases = "NEW_TYPE", []string{name}
		if getProviderDirectory(&renamed) != directory || providerDocSlug(&renamed) != providerDocSlug(def) {
			t.Fatalf("%s: type rename changed source or documentation paths", name)
		}
	}
	for _, def := range publicDefinitions() {
		if strings.HasPrefix(def.TypeName, "GENERATOR_TEST_") {
			continue
		}
		if _, err := os.Stat(filepath.Join("../../documentation/provider", providerDocSlug(def)+".md")); err != nil {
			t.Errorf("%s documentation: %v", def.TypeName, err)
		}
		if _, err := os.Stat(filepath.Join("../../providers", getProviderDirectory(def))); err != nil {
			t.Errorf("%s implementation directory: %v", def.TypeName, err)
		}
	}
	matrix := matrixData()
	if _, found := matrix.Providers["GENERATOR_TEST_A_ALIAS"]; found {
		t.Fatal("feature matrix enumerated an alias")
	}
	if _, found := matrix.Providers["GENERATOR_TEST_A"]; !found {
		t.Fatal("feature matrix omitted a canonical definition")
	}
}
