package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	_ "github.com/DNSControl/dnscontrol/v5/pkg/providers/_all"
	"github.com/fbiville/markdown-table-formatter/pkg/markdown"
)

func generateFeatureMatrix() error {
	updateProviderDocs()

	var replacementContent strings.Builder
	replacementContent.WriteString("Jump to a section:\n\n")
	matrix := matrixData()

	for _, table := range matrix.FeatureTables {
		var anchor = strings.ToLower(table.Title)
		anchor = strings.ReplaceAll(anchor, " ", "-")

		fmt.Fprintf(&replacementContent, "- [%s](#%s)\n", table.Title, anchor)
	}

	for i, table := range matrix.FeatureTables {
		fmt.Fprintf(&replacementContent, "\n### %s <!--(section %d/%d)-->\n\n",
			table.Title, i+1, len(matrix.FeatureTables))
		var tableContent string
		if table.ListStyle {
			tableContent = recordTypesSection(matrix, table)
		} else {
			var err error
			tableContent, err = markdownTable(matrix, table)
			if err != nil {
				return err
			}
		}
		replacementContent.WriteString(tableContent)
		replacementContent.WriteString("\n")
	}

	replaceInlineContent(
		"documentation/provider/index.md",
		"<!-- provider-matrix-start -->",
		"<!-- provider-matrix-end -->",
		replacementContent.String(),
	)

	return nil
}

func updateProviderDocs() {
	matrix := matrixData()
	docDir := "documentation/provider"
	for _, def := range publicDefinitions() {
		providerName := def.TypeName
		docFile := filepath.Join(docDir, providerDocSlug(def)+".md")
		if _, err := os.Stat(docFile); os.IsNotExist(err) {
			fmt.Printf("WARNING: Missing documentation page for provider %s: %s\n", providerName, docFile)
		} else {
			featureList := buildFeatureList(matrix, providerName)
			replaceInlineContent(
				docFile,
				"<!-- provider-features-start -->",
				"<!-- provider-features-end -->",
				featureList,
			)
		}
	}
}

func buildFeatureList(matrix *FeatureMatrix, providerName string) string {
	var sb strings.Builder
	featureMap := matrix.Providers[providerName]

	for _, category := range matrix.FeatureTables {
		if category.ListStyle {
			supported, unimplemented, notSupported := classifyRecordTypes(featureMap, category.Features)
			if len(supported) == 0 && len(unimplemented) == 0 && len(notSupported) == 0 {
				continue // no documented record-type support
			}
			sb.WriteString("- ")
			sb.WriteString(category.Title)
			sb.WriteString("\n  - ")
			sb.WriteString(recordTypeList(supported))
			sb.WriteString("\n")
			writeRecordTypeAnnotations(&sb, featureMap, "    - ", "Unimplemented", unimplemented)
			writeRecordTypeAnnotations(&sb, featureMap, "    - ", "Not supported", notSupported)
			continue
		}

		sb.WriteString("- ")
		sb.WriteString(category.Title)
		sb.WriteString("\n")

		for _, feature := range category.Features {
			emoji := featureEmoji(featureMap, feature)
			sb.WriteString("  - ")
			sb.WriteString(feature)
			sb.WriteString(": ")
			sb.WriteString(emoji)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// recordTypesNote is appended to the record types list, explaining its
// annotations.
const recordTypesNote = `NOTE: "Unimplemented" means that the provider's API supports this record type but DNSControl's implementation does not (yet) implement it. "Not supported" means that the record type is not supported and the reason has been documented. A record type that is not listed is simply not supported.`

// recordTypesSection renders the record types as a bullet list: one bullet
// per provider naming its supported record types as a sorted list of links,
// with annotated record types as sub-bullets. Providers without any
// documented record-type support are omitted.
func recordTypesSection(matrix *FeatureMatrix, table FeatureTableDef) string {
	var sb strings.Builder
	for _, def := range publicDefinitions() {
		featureMap := matrix.Providers[def.TypeName]
		supported, unimplemented, notSupported := classifyRecordTypes(featureMap, table.Features)
		if len(supported) == 0 && len(unimplemented) == 0 && len(notSupported) == 0 {
			continue // no documented record-type support
		}
		fmt.Fprintf(&sb, "- [`%s`](%s.md): %s\n",
			def.TypeName, providerDocSlug(def), recordTypeList(supported))
		writeRecordTypeAnnotations(&sb, featureMap, "  - ", "Unimplemented", unimplemented)
		writeRecordTypeAnnotations(&sb, featureMap, "  - ", "Not supported", notSupported)
	}
	fmt.Fprintf(&sb, "\n%s\n", recordTypesNote)
	return sb.String()
}

// classifyRecordTypes buckets one provider's record types into the supported
// list and the annotated Unimplemented and Not-supported lists, each sorted
// by record type name. Record types with a nil or unannotated note are
// omitted: a record type that is not listed is simply not supported.
func classifyRecordTypes(featureMap FeatureMap, featureNames []string) (supported, unimplemented, notSupported []string) {
	for _, featureName := range featureNames {
		note := featureMap[featureName]
		if note == nil || (!note.HasFeature && !note.Unimplemented && note.Comment == "" && note.Link == "") {
			continue
		}
		switch {
		case note.HasFeature:
			supported = append(supported, featureName)
		case note.Unimplemented:
			unimplemented = append(unimplemented, featureName)
		default:
			// An annotated "Cannot": the reason deserves a sub-bullet.
			notSupported = append(notSupported, featureName)
		}
	}
	sortRecordTypes(supported)
	sortRecordTypes(unimplemented)
	sortRecordTypes(notSupported)
	return supported, unimplemented, notSupported
}

// sortRecordTypes sorts record-type links by their bare type names. The
// links themselves do not sort correctly: "[`A`](.../A.md)" sorts after
// "[`AAAA`](...)" because ']' > 'A'.
func sortRecordTypes(features []string) {
	slices.SortFunc(features, func(a, b string) int {
		return strings.Compare(recordTypeName(a), recordTypeName(b))
	})
}

// recordTypeName extracts the bare type name from a record-type feature
// link: "[`SOA`](../language-reference/domain-modifiers/SOA.md)" -> "SOA".
func recordTypeName(feature string) string {
	name, _, _ := strings.Cut(strings.TrimPrefix(feature, "[`"), "`]")
	return name
}

// recordTypeList joins record-type links into one space-separated list.
func recordTypeList(features []string) string {
	if len(features) == 0 {
		return "none"
	}
	return strings.Join(features, " ")
}

// writeRecordTypeAnnotations renders annotated record types as sub-bullets,
// e.g. "  * Unimplemented: [`SOA`](...) (reason)".
func writeRecordTypeAnnotations(sb *strings.Builder, featureMap FeatureMap, prefix, label string, features []string) {
	for _, feature := range features {
		note := featureMap[feature]
		var detail string
		switch {
		case note == nil:
		case note.Comment != "" && note.Link != "":
			detail = " (" + note.Comment + ": " + note.Link + ")"
		case note.Comment != "":
			detail = " (" + note.Comment + ")"
		case note.Link != "":
			detail = " (" + note.Link + ")"
		}
		fmt.Fprintf(sb, "%s%s: %s%s\n", prefix, label, feature, detail)
	}
}

func markdownTable(matrix *FeatureMatrix, table FeatureTableDef) (string, error) {
	var tableHeaders []string
	tableHeaders = append(tableHeaders, "Provider name")
	tableHeaders = append(tableHeaders, table.Features...)

	var tableData [][]string
	for _, def := range publicDefinitions() {
		providerName := def.TypeName
		featureMap := matrix.Providers[providerName]

		providerLink := providerDocSlug(def)

		var tableDataRow []string
		tableDataRow = append(tableDataRow, "[`"+providerName+"`]("+providerLink+".md)")
		for _, featureName := range table.Features {
			tableDataRow = append(tableDataRow, featureEmoji(featureMap, featureName))
		}
		skipThisRow := true
		for status := range tableDataRow[1:] {
			if tableDataRow[status+1] != "❔" {
				skipThisRow = false
			}
		}
		if !skipThisRow {
			tableData = append(tableData, tableDataRow)
		}
	}

	markdownTable, err := markdown.NewTableFormatterBuilder().
		Build(tableHeaders...).
		Format(tableData)
	if err != nil {
		return "", err
	}

	return markdownTable, nil
}

func featureEmoji(
	featureMap FeatureMap,
	featureName string,
) string {
	if featureMap[featureName] == nil {
		return "❔"
	}

	if featureMap[featureName].HasFeature {
		return "✅"
	} else if featureMap[featureName].Unimplemented {
		return "❔"
	}
	return "❌"
}

func matrixData() *FeatureMatrix {
	const (
		OfficialSupport          = "[Official Support](../provider/index.md#providers-with-official-support)" // vs. community supported
		ProviderDNSProvider      = "DNS Provider"
		ProviderRegistrar        = "Registrar"
		ProviderThreadSafe       = "[Concurrent fetching](../advanced-features/concurrency-verified.md)"
		DomainModifierA          = "[`A`](../language-reference/domain-modifiers/A.md)"
		DomainModifierAaaa       = "[`AAAA`](../language-reference/domain-modifiers/AAAA.md)"
		DomainModifierAlias      = "[`ALIAS`](../language-reference/domain-modifiers/ALIAS.md)"
		DomainModifierCaa        = "[`CAA`](../language-reference/domain-modifiers/CAA.md)"
		DomainModifierCname      = "[`CNAME`](../language-reference/domain-modifiers/CNAME.md)"
		DomainModifierCsync      = "[`CSYNC`](../language-reference/domain-modifiers/CSYNC.md)"
		DomainModifierDhcid      = "[`DHCID`](../language-reference/domain-modifiers/DHCID.md)"
		DomainModifierDname      = "[`DNAME`](../language-reference/domain-modifiers/DNAME.md)"
		DomainModifierDnskey     = "[`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md)"
		FeatureAutoDNSSEC        = "[`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md)"
		DomainModifierDs         = "[`DS`](../language-reference/domain-modifiers/DS.md)"
		DomainModifierHTTPS      = "[`HTTPS`](../language-reference/domain-modifiers/HTTPS.md)"
		DomainModifierLoc        = "[`LOC`](../language-reference/domain-modifiers/LOC.md)"
		DomainModifierMx         = "[`MX`](../language-reference/domain-modifiers/MX.md)"
		DomainModifierNaptr      = "[`NAPTR`](../language-reference/domain-modifiers/NAPTR.md)"
		DomainModifierNs         = "[`NS`](../language-reference/domain-modifiers/NS.md)"
		DomainModifierOpenpgpkey = "[`OPENPGPKEY`](../language-reference/domain-modifiers/OPENPGPKEY.md)"
		DomainModifierPtr        = "[`PTR`](../language-reference/domain-modifiers/PTR.md)"
		DomainModifierRP         = "[`RP`](../language-reference/domain-modifiers/RP.md)"
		DomainModifierSMIMEA     = "[`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md)"
		DomainModifierSoa        = "[`SOA`](../language-reference/domain-modifiers/SOA.md)"
		DomainModifierSrv        = "[`SRV`](../language-reference/domain-modifiers/SRV.md)"
		DomainModifierSshfp      = "[`SSHFP`](../language-reference/domain-modifiers/SSHFP.md)"
		DomainModifierSvcb       = "[`SVCB`](../language-reference/domain-modifiers/SVCB.md)"
		DomainModifierTlsa       = "[`TLSA`](../language-reference/domain-modifiers/TLSA.md)"
		DomainModifierTxt        = "[`TXT`](../language-reference/domain-modifiers/TXT.md)"
		DualHost                 = "[dual host](../advanced-features/dual-host.md)"
		CreateDomains            = "create-domains"
		GetZones                 = "[get-zones](../commands/get-zones.md)"
	)

	matrix := &FeatureMatrix{
		Providers: map[string]FeatureMap{},
		FeatureTables: []FeatureTableDef{
			{ // provider type
				Title: "Provider Type",
				Features: []string{
					OfficialSupport,
					ProviderDNSProvider,
					ProviderRegistrar,
				},
			},
			{ // provider API
				Title: "Provider API",
				Features: []string{
					ProviderThreadSafe,
					DualHost,
					CreateDomains,
					// NoPurge,
					GetZones,
					FeatureAutoDNSSEC,
				},
			},
			{ // record types, rendered as a per-provider bullet list
				Title:     "Record types",
				ListStyle: true,
				Features: []string{
					DomainModifierA,
					DomainModifierAaaa,
					DomainModifierAlias,
					DomainModifierCaa,
					DomainModifierCname,
					DomainModifierCsync,
					DomainModifierDhcid,
					DomainModifierDname,
					DomainModifierDnskey,
					DomainModifierDs,
					DomainModifierHTTPS,
					DomainModifierLoc,
					DomainModifierMx,
					DomainModifierNaptr,
					DomainModifierNs,
					DomainModifierOpenpgpkey,
					DomainModifierPtr,
					DomainModifierRP,
					DomainModifierSMIMEA,
					DomainModifierSoa,
					DomainModifierSrv,
					DomainModifierSshfp,
					DomainModifierSvcb,
					DomainModifierTlsa,
					DomainModifierTxt,
				},
			},
		},
	}

	for _, def := range publicDefinitions() {
		providerName := def.TypeName
		featureMap := FeatureMap{}

		// setRecordType reports support for a record type from the provider's
		// SupportedTypes, including reason and link annotations.
		setRecordType := func(
			featureName string,
			typeName string,
		) {
			note := def.RecordTypeSupport(typeName)
			featureMap[featureName] = &note
		}

		if def.OfficiallySupported != nil {
			featureMap[OfficialSupport] = def.OfficiallySupported
		} else {
			featureMap[OfficialSupport] = providers.Cannot()
		}
		featureMap.SetSimple(
			ProviderDNSProvider,
			false,
			func() bool { return def.Kind.Has(providers.KindDNS) },
		)
		featureMap.SetSimple(
			ProviderRegistrar,
			false,
			func() bool { return def.Kind.Has(providers.KindRegistrar) },
		)
		featureMap[ProviderThreadSafe] = def.CanConcur
		featureMap[FeatureAutoDNSSEC] = def.CanAutoDNSSEC
		setRecordType(DomainModifierA, "A")
		setRecordType(DomainModifierAaaa, "AAAA")
		setRecordType(DomainModifierAlias, "ALIAS")
		setRecordType(DomainModifierCaa, "CAA")
		setRecordType(DomainModifierCname, "CNAME")
		setRecordType(DomainModifierCsync, "CSYNC")
		setRecordType(DomainModifierDhcid, "DHCID")
		setRecordType(DomainModifierDname, "DNAME")
		setRecordType(DomainModifierDnskey, "DNSKEY")
		setRecordType(DomainModifierDs, "DS")
		setRecordType(DomainModifierHTTPS, "HTTPS")
		setRecordType(DomainModifierLoc, "LOC")
		setRecordType(DomainModifierMx, "MX")
		setRecordType(DomainModifierNaptr, "NAPTR")
		setRecordType(DomainModifierNs, "NS")
		setRecordType(DomainModifierOpenpgpkey, "OPENPGPKEY")
		setRecordType(DomainModifierPtr, "PTR")
		setRecordType(DomainModifierRP, "RP")
		setRecordType(DomainModifierSMIMEA, "SMIMEA")
		setRecordType(DomainModifierSoa, "SOA")
		setRecordType(DomainModifierSrv, "SRV")
		setRecordType(DomainModifierSshfp, "SSHFP")
		setRecordType(DomainModifierSvcb, "SVCB")
		setRecordType(DomainModifierTlsa, "TLSA")
		setRecordType(DomainModifierTxt, "TXT")
		featureMap[GetZones] = &providers.DocumentationNote{HasFeature: def.CanGetZones, Comment: def.CanGetZonesReason}
		featureMap[CreateDomains] = &providers.DocumentationNote{HasFeature: def.DocCreateDomains, Comment: def.CreateDomainsReason}
		featureMap[DualHost] = def.CanDualHost
		matrix.Providers[providerName] = featureMap
	}
	return matrix
}

// FeatureMap maps provider names to compliance documentation.
type FeatureMap map[string]*providers.DocumentationNote

// SetSimple configures a provider's setting in featureMap.
func (featureMap FeatureMap) SetSimple(
	name string,
	unknownsAllowed bool,
	f func() bool,
) {
	if f() {
		featureMap[name] = &providers.DocumentationNote{HasFeature: true}
	} else if !unknownsAllowed {
		featureMap[name] = &providers.DocumentationNote{HasFeature: false}
	}
}

// FeatureMatrix describes features and which providers support it.
type FeatureMatrix struct {
	Providers map[string]FeatureMap
	// FeatureTables renders in order. Tables with ListStyle render one bullet
	// per provider, listing the provider's record types with annotated types
	// as sub-bullets; the rest render one emoji column per feature.
	FeatureTables []FeatureTableDef
}

// FeatureTableDef is one section of the provider feature matrix.
type FeatureTableDef struct {
	Title     string
	Features  []string
	ListStyle bool
}

func replaceInlineContent(
	file string,
	startMarker string,
	endMarker string,
	newContent string,
) {
	contentBytes, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	content := string(contentBytes)

	start := strings.Index(content, startMarker)
	end := strings.Index(content, endMarker)

	if start == -1 || end == -1 {
		fmt.Printf("WARNING: Missing delimiters in %s (start=%t, end=%t), skipping\n",
			file, start != -1, end != -1)
		return
	}

	newContentString := startMarker + "\n" + newContent + endMarker
	newContentBytes := []byte(newContentString)
	contentBytes = []byte(content)
	contentBytes = append(contentBytes[:start], append(newContentBytes, contentBytes[end+len(endMarker):]...)...)

	err = os.WriteFile(file, contentBytes, 0o644)
	if err != nil {
		panic(err)
	}
}
