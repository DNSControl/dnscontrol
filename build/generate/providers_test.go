package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
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
	if matrix.FeatureTables[0].Title != "Provider Support Level and Type" || matrix.FeatureTables[1].Title != "Provider API-level Features" {
		t.Fatalf("unexpected table titles: %q, %q", matrix.FeatureTables[0].Title, matrix.FeatureTables[1].Title)
	}
	apiTable := matrix.FeatureTables[1]
	autoDnssec := "[AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md)"
	if !slices.Contains(apiTable.Features, autoDnssec) {
		t.Fatal("AUTODNSSEC missing from the Provider API table")
	}
	recordTypes := matrix.FeatureTables[2]
	if recordTypes.Title != "Supported Record Types by Provider" ||
		recordTypes.PageTitle != "Supported Record Types" || !recordTypes.ListStyle {
		t.Fatalf("table 3 must be the list-style record types table, got %+v", recordTypes)
	}
	if slices.Contains(recordTypes.Features, autoDnssec) {
		t.Fatal("AUTODNSSEC must not be listed as a record type")
	}
	// The record types are derived from the record catalog: every standard
	// type that dnsconfig.js can create, plus ALIAS. Each must link to the
	// type's documentation page.
	for _, feature := range recordTypes.Features {
		name := recordTypeName(feature)
		if feature != recordTypeFeature(name) {
			t.Fatalf("%s: feature link does not match recordTypeFeature", feature)
		}
		typ, ok := privatetypes.LookupRecordType(name)
		if !ok {
			t.Fatalf("%s: not a catalog record type", name)
		}
		if typ.Pseudo && name != "ALIAS" {
			t.Fatalf("%s: provider-specific pseudo type listed in the matrix", name)
		}
		if !privatetypes.HasMaker(name) {
			t.Fatalf("%s: listed without a Make function", name)
		}
		if _, err := os.Stat(filepath.Join("../../documentation/language-reference/domain-modifiers", name+".md")); err != nil {
			t.Fatalf("%s: record types must link to an existing docs page: %v", name, err)
		}
	}
	if !slices.IsSortedFunc(recordTypes.Features, func(a, b string) int {
		return strings.Compare(recordTypeName(a), recordTypeName(b))
	}) {
		t.Fatal("record types are not sorted by type name")
	}
	// ALIAS is the one pseudo type listed: many providers implement it.
	if !slices.ContainsFunc(recordTypes.Features, func(f string) bool { return recordTypeName(f) == "ALIAS" }) {
		t.Fatal("ALIAS missing from the record types")
	}
	// Every Basic8 record type must be listed.
	for _, typ := range []string{"A", "AAAA", "CAA", "CNAME", "MX", "NS", "SRV", "TXT"} {
		found := false
		for _, feature := range recordTypes.Features {
			if recordTypeName(feature) == typ {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Basic8 record type %s is missing from the matrix", typ)
		}
	}
	// The buckets must partition the features without duplicates.
	for name, featureMap := range matrix.Providers {
		supported, unimplemented, notSupported := classifyRecordTypes(featureMap, recordTypes.Features)
		seen := make(map[string]bool, len(recordTypes.Features))
		for _, bucket := range [][]string{supported, unimplemented, notSupported} {
			for _, feature := range bucket {
				if !slices.Contains(recordTypes.Features, feature) {
					t.Fatalf("%s: record-type bucket contains unknown feature %q", name, feature)
				}
				if seen[feature] {
					t.Fatalf("%s: record type %q appears in two buckets", name, feature)
				}
				seen[feature] = true
			}
		}
	}
}

func TestClassifyRecordTypes(t *testing.T) {
	aaaa := "[AAAA](../language-reference/domain-modifiers/AAAA.md)"
	alias := "[ALIAS](../language-reference/domain-modifiers/ALIAS.md)"
	caa := "[CAA](../language-reference/domain-modifiers/CAA.md)"
	cname := "[CNAME](../language-reference/domain-modifiers/CNAME.md)"
	loc := "[LOC](../language-reference/domain-modifiers/LOC.md)"
	soa := "[SOA](../language-reference/domain-modifiers/SOA.md)"
	srv := "[SRV](../language-reference/domain-modifiers/SRV.md)"

	// Deliberately unsorted input.
	features := []string{cname, alias, srv, caa, aaaa, soa, loc}
	featureMap := FeatureMap{
		aaaa:  providers.Can(),
		alias: providers.Can(),
		caa:   providers.Can(),
		cname: providers.Can(),
		srv:   providers.Can(),
		loc:   &providers.DocumentationNote{Comment: "API cannot do LOC"}, // annotated Cannot
		soa:   providers.Unimplemented("Awaiting API support"),
	}

	supported, unimplemented, notSupported := classifyRecordTypes(featureMap, features)

	// Supported types are sorted by type name, not by link.
	wantSupported := []string{aaaa, alias, caa, cname, srv}
	if !slices.Equal(supported, wantSupported) {
		t.Fatalf("supported = %v, want %v", supported, wantSupported)
	}
	if !slices.Equal(unimplemented, []string{soa}) {
		t.Fatalf("unimplemented = %v, want [%s]", unimplemented, soa)
	}
	if !slices.Equal(notSupported, []string{loc}) {
		t.Fatalf("notSupported = %v, want [%s]", notSupported, loc)
	}

	// A zero-value note and a nil note are omitted, not listed.
	featureMap[caa] = &providers.DocumentationNote{}
	delete(featureMap, srv)
	supported, unimplemented, notSupported = classifyRecordTypes(featureMap, features)
	if !slices.Equal(supported, []string{aaaa, alias, cname}) {
		t.Fatalf("supported = %v, want [%s %s %s]", supported, aaaa, alias, cname)
	}
	if len(unimplemented) != 1 || len(notSupported) != 1 {
		t.Fatalf("annotation buckets changed size: %v %v", unimplemented, notSupported)
	}
}

func TestRecordTypeList(t *testing.T) {
	if got := recordTypeList(nil); got != "none" {
		t.Fatalf("empty list renders as %q, want %q", got, "none")
	}
	a, b := "[A](a.md)", "[B](b.md)"
	if got := recordTypeList([]string{a, b}); got != a+" "+b {
		t.Fatalf("list renders as %q", got)
	}
}

func TestRecordTypeName(t *testing.T) {
	if got := recordTypeName("[SOA](../language-reference/domain-modifiers/SOA.md)"); got != "SOA" {
		t.Fatalf("recordTypeName = %q, want SOA", got)
	}
}

func TestRecordTypeAnnotations(t *testing.T) {
	loc := "[LOC](../language-reference/domain-modifiers/LOC.md)"
	featureMap := FeatureMap{loc: providers.Unimplemented("buggy API")}
	var sb strings.Builder
	writeRecordTypeAnnotations(&sb, featureMap, "  - ", "Unimplemented", []string{loc})
	want := "  - Unimplemented: " + loc + " (buggy API)\n"
	if sb.String() != want {
		t.Fatalf("annotation rendered as %q, want %q", sb.String(), want)
	}

	sb.Reset()
	featureMap[loc] = providers.Unimplemented("buggy API", "https://example.test/issue")
	writeRecordTypeAnnotations(&sb, featureMap, "    - ", "Not supported", []string{loc})
	want = "    - Not supported: " + loc + " (buggy API: https://example.test/issue)\n"
	if sb.String() != want {
		t.Fatalf("annotation rendered as %q, want %q", sb.String(), want)
	}

	sb.Reset()
	featureMap[loc] = providers.Unimplemented()
	writeRecordTypeAnnotations(&sb, featureMap, "  - ", "Unimplemented", []string{loc})
	want = "  - Unimplemented: " + loc + "\n"
	if sb.String() != want {
		t.Fatalf("annotation rendered as %q, want %q", sb.String(), want)
	}
}

func TestGenerateFeatureMatrixRendersRecordTypesList(t *testing.T) {
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
		"- [Provider Support Level and Type](#provider-support-level-and-type)",
		"- [Provider API-level Features](#provider-api-level-features)",
		"- [Supported Record Types by Provider](#supported-record-types-by-provider)",
		"### Provider Support Level and Type <!--(section 1/3)-->",
		"### Provider API-level Features <!--(section 2/3)-->",
		"### Supported Record Types by Provider <!--(section 3/3)-->",
		"[AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md)",
		`NOTE: "Unimplemented" means`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated matrix is missing %q", want)
		}
	}
	for _, stale := range []string{
		"### DNS extensions", "### Service discovery", "### Security", "### DNSSEC",
		"| Provider name | Supported | Unsupported | Unknown |",
	} {
		if strings.Contains(got, stale) {
			t.Errorf("generated matrix still contains stale table %q", stale)
		}
	}

	section := got[strings.Index(got, "### Supported Record Types by Provider"):strings.Index(got, "<!-- provider-matrix-end -->")]

	// The NOTE defines the annotations, so it must sit above the list.
	if !strings.HasPrefix(section, "### Supported Record Types by Provider <!--(section 3/3)-->\n\nNOTE: \"Unimplemented\" means") {
		t.Error("record types section must start with the NOTE above the list")
	}

	// The section is a bullet list, not a table.
	for line := range strings.SplitSeq(section, "\n") {
		if strings.HasPrefix(line, "|") {
			t.Errorf("record types section contains a table row: %s", line)
		}
	}

	// The GENERATOR_TEST providers declare no SupportedTypes, so they default
	// to Basic8: all eight basic record types render, sorted by type name.
	var basic8 strings.Builder
	basic8.WriteString("- [GENERATOR_TEST_A](GENERATOR_TEST_A.md):")
	for _, typ := range []string{"A", "AAAA", "CAA", "CNAME", "MX", "NS", "SRV", "TXT"} {
		fmt.Fprintf(&basic8, " [%s](../language-reference/domain-modifiers/%s.md)", typ, typ)
	}
	if !strings.Contains(section, basic8.String()) {
		t.Errorf("Basic8 providers should render %q", basic8.String())
	}

	// Real providers exercise the annotation sub-bullets: CLOUDNS annotates
	// SOA as Unimplemented and DS as Cannot.
	wantUnimplemented := "  - Unimplemented: [SOA](../language-reference/domain-modifiers/SOA.md)" +
		" (Supported by cloudns at a separate API endpoint (/dns/modify-soa.json), not implemented yet)"
	if !strings.Contains(section, wantUnimplemented) {
		t.Errorf("CLOUDNS SOA sub-bullet missing:\n%s", wantUnimplemented)
	}
	wantNotSupported := "  - Not supported: [DS](../language-reference/domain-modifiers/DS.md)" +
		" (Not supported for root, only for children)"
	if !strings.Contains(section, wantNotSupported) {
		t.Errorf("CLOUDNS DS sub-bullet missing:\n%s", wantNotSupported)
	}

	// Providers without any documented record support are skipped.
	if strings.Contains(section, "- [DYNADOT](") {
		t.Error("DYNADOT has no record support and must not appear in the record types list")
	}
}

func TestBuildFeatureListRecordTypes(t *testing.T) {
	matrix := matrixData()

	cloudns := buildFeatureList(matrix, "CLOUDNS")
	if !strings.Contains(cloudns, "- Supported Record Types\n  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md)") {
		t.Error("CLOUDNS record types should begin with the Basic8 list")
	}
	if !strings.Contains(cloudns, "    - Unimplemented: [SOA](../language-reference/domain-modifiers/SOA.md) (Supported by cloudns at a separate API endpoint (/dns/modify-soa.json), not implemented yet)") {
		t.Error("CLOUDNS record types should include the SOA annotation sub-bullet")
	}
	if !strings.Contains(cloudns, "    - Not supported: [DS](../language-reference/domain-modifiers/DS.md) (Not supported for root, only for children)") {
		t.Error("CLOUDNS record types should include the DS annotation sub-bullet")
	}
	// The record types category no longer renders per-type emoji lines.
	recordTypes := cloudns[strings.Index(cloudns, "- Supported Record Types"):]
	if strings.Contains(recordTypes, ": ✅") || strings.Contains(recordTypes, ": ❌") || strings.Contains(recordTypes, ": ❔") {
		t.Error("record types category still renders per-type emoji lines")
	}

	dynadot := buildFeatureList(matrix, "DYNADOT")
	if strings.Contains(dynadot, "- Supported Record Types") {
		t.Error("DYNADOT has no record support; its provider page should omit the record types category")
	}
}
