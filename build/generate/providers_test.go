package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

func TestFeatureMatrixStructure(t *testing.T) {
	matrix := matrixData()
	if len(matrix.FeatureTables) != 3 {
		t.Fatalf("got %d tables, want 3", len(matrix.FeatureTables))
	}
	if matrix.FeatureTables[0].Title != "Provider Type" || matrix.FeatureTables[1].Title != "Provider API" {
		t.Fatalf("unexpected table titles: %q, %q", matrix.FeatureTables[0].Title, matrix.FeatureTables[1].Title)
	}
	apiTable := matrix.FeatureTables[1]
	autoDnssec := "[`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md)"
	if !slices.Contains(apiTable.Features, autoDnssec) {
		t.Fatal("AUTODNSSEC missing from the Provider API table")
	}
	recordTypes := matrix.FeatureTables[2]
	if recordTypes.Title != "Record types" || !recordTypes.ListStyle {
		t.Fatalf("table 3 must be the list-style Record types table, got %+v", recordTypes)
	}
	if slices.Contains(recordTypes.Features, autoDnssec) {
		t.Fatal("AUTODNSSEC must not be listed as a record type")
	}
	if len(recordTypes.Features) != 19 {
		t.Fatalf("got %d record types, want 19", len(recordTypes.Features))
	}
	if !slices.IsSorted(recordTypes.Features) {
		t.Fatal("record types are not in alphabetical order")
	}
	for name, featureMap := range matrix.Providers {
		supported, unsupported, unknownCount := bucketRecordTypes(featureMap, recordTypes.Features)
		if got := len(supported) + len(unsupported) + unknownCount; got != len(recordTypes.Features) {
			t.Fatalf("%s: record-type buckets cover %d features, want %d", name, got, len(recordTypes.Features))
		}
	}
}

func TestBucketRecordTypesPartitionsFeatures(t *testing.T) {
	const alias = "[`ALIAS`](../language-reference/domain-modifiers/ALIAS.md)"
	const caa = "[`CAA`](../language-reference/domain-modifiers/CAA.md)"
	const srv = "[`SRV`](../language-reference/domain-modifiers/SRV.md)"
	features := []string{alias, caa, srv}
	featureMap := FeatureMap{
		alias: &providers.DocumentationNote{HasFeature: true},
		caa:   &providers.DocumentationNote{}, // documented as unsupported
		// srv absent: a nil note is unknown
	}

	supported, unsupported, unknownCount := bucketRecordTypes(featureMap, features)

	if len(supported)+len(unsupported)+unknownCount != len(features) {
		t.Fatalf("buckets do not cover all features: %d + %d + %d != %d",
			len(supported), len(unsupported), unknownCount, len(features))
	}
	if len(supported) != 1 || supported[0] != alias {
		t.Fatalf("supported = %v, want [%s]", supported, alias)
	}
	if len(unsupported) != 1 || unsupported[0] != caa {
		t.Fatalf("unsupported = %v, want [%s]", unsupported, caa)
	}
	if unknownCount != 1 {
		t.Fatalf("unknownCount = %d, want 1 (nil note)", unknownCount)
	}

	// An Unimplemented note counts as unknown too.
	featureMap[srv] = providers.Unimplemented()
	_, _, unknownCount = bucketRecordTypes(featureMap, features)
	if unknownCount != 1 {
		t.Fatalf("unknownCount = %d, want 1 (Unimplemented note)", unknownCount)
	}
}

func TestListWithCount(t *testing.T) {
	if got := listWithCount(nil); got != "0" {
		t.Fatalf("empty list renders as %q, want %q", got, "0")
	}
	a, b := "[`A`](a.md)", "[`B`](b.md)"
	if got := listWithCount([]string{a, b}); got != "2: "+a+", "+b {
		t.Fatalf("list cell renders as %q", got)
	}
}

func TestGenerateFeatureMatrixRendersCombinedRecordTable(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("documentation/provider", 0o755); err != nil {
		t.Fatal(err)
	}
	const before = "Intro\n<!-- provider-matrix-start -->\nold matrix\n<!-- provider-matrix-end -->\nOutro\n"
	if err := os.WriteFile("documentation/provider/index.md", []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := generateFeatureMatrix(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("documentation/provider/index.md")
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.HasPrefix(got, "Intro\n<!-- provider-matrix-start -->") ||
		!strings.HasSuffix(got, "<!-- provider-matrix-end -->\nOutro\n") {
		t.Fatal("generation changed content outside the managed section")
	}

	for _, want := range []string{
		"- [Provider Type](#provider-type)",
		"- [Provider API](#provider-api)",
		"- [Record types](#record-types)",
		"### Provider Type <!--(table 1/3)-->",
		"### Provider API <!--(table 2/3)-->",
		"### Record types <!--(table 3/3)-->",
		"| Provider name | Supported | Unsupported | Unknown |",
		"[`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated matrix is missing %q", want)
		}
	}
	for _, stale := range []string{
		"### DNS extensions", "### Service discovery", "### Security", "### DNSSEC",
	} {
		if strings.Contains(got, stale) {
			t.Errorf("generated matrix still contains stale table %q", stale)
		}
	}

	// The GENERATOR_TEST providers declare no SupportedTypes, so they default
	// to Basic8: CAA and SRV land in the supported cell, in matrix order.
	const basic8Supported = "2: [`CAA`](../language-reference/domain-modifiers/CAA.md), [`SRV`](../language-reference/domain-modifiers/SRV.md)"
	if !strings.Contains(got, basic8Supported) {
		t.Errorf("Basic8 providers should render %q as their supported cell", basic8Supported)
	}

	// Every record-types row must be a 4-column table row.
	section := got[strings.Index(got, "### Record types"):strings.Index(got, "<!-- provider-matrix-end -->")]
	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "| [`") {
			continue
		}
		if strings.Count(line, "|") != 5 {
			t.Errorf("record-types row does not have 4 cells: %s", line)
		}
	}
}
