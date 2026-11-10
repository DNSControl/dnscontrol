package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	_ "github.com/DNSControl/dnscontrol/v5/pkg/providers/_all"
	"github.com/fbiville/markdown-table-formatter/pkg/markdown"
)

func generateFeatureMatrix() error {
	updateProviderDocs()

	var replacementContent strings.Builder
	replacementContent.WriteString("Jump to a table:\n\n")
	matrix := matrixData()

	for _, table := range matrix.FeatureTables {
		var anchor = strings.ToLower(table.Title)
		anchor = strings.ReplaceAll(anchor, " ", "-")

		replacementContent.WriteString(fmt.Sprintf("- [%s](#%s)\n", table.Title, anchor))
	}

	for i, table := range matrix.FeatureTables {
		fmt.Fprintf(&replacementContent, "\n### %s <!--(table %d/%d)-->\n\n",
			table.Title, i+1, len(matrix.FeatureTables))
		var tableContent string
		var err error
		if table.ListStyle {
			tableContent = recordTypesTable(matrix, table)
		} else {
			tableContent, err = markdownTable(matrix, table)
		}
		if err != nil {
			return err
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

// recordTypesTable renders the combined record-types table: one row per
// provider, with the supported and unsupported record types as compact lists
// (each prefixed with its count) and a count-only column for unknown
// statuses. The table is hand-rolled instead of using the markdown-table
// formatter, which would pad every column to the width of the widest
// (19-link) cell.
func recordTypesTable(matrix *FeatureMatrix, table FeatureTableDef) string {
	var sb strings.Builder
	sb.WriteString("| Provider name | Supported | Unsupported | Unknown |\n")
	sb.WriteString("| ------------- | --------- | ----------- | ------- |\n")

	for _, def := range publicDefinitions() {
		featureMap := matrix.Providers[def.TypeName]
		supported, unsupported, unknownCount := bucketRecordTypes(featureMap, table.Features)
		if len(supported) == 0 && len(unsupported) == 0 {
			continue // no documented record-type support at all
		}
		providerLink := providerDocSlug(def)
		fmt.Fprintf(&sb, "| [`%s`](%s.md) | %s | %s | %d |\n",
			def.TypeName, providerLink, listWithCount(supported), listWithCount(unsupported), unknownCount)
	}

	return sb.String()
}

// bucketRecordTypes splits features into the supported and unsupported lists,
// plus a count of the features whose status is unknown. It mirrors the
// featureEmoji semantics: a nil note or an Unimplemented note is unknown.
func bucketRecordTypes(featureMap FeatureMap, featureNames []string) (supported, unsupported []string, unknownCount int) {
	for _, featureName := range featureNames {
		note := featureMap[featureName]
		switch {
		case note == nil || note.Unimplemented:
			unknownCount++
		case note.HasFeature:
			supported = append(supported, featureName)
		default:
			unsupported = append(unsupported, featureName)
		}
	}
	return supported, unsupported, unknownCount
}

// listWithCount renders a record-type list cell: "N: item1, item2, ...".
// An empty list renders as just the count.
func listWithCount(items []string) string {
	if len(items) == 0 {
		return "0"
	}
	return fmt.Sprintf("%d: %s", len(items), strings.Join(items, ", "))
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
		DomainModifierAlias      = "[`ALIAS`](../language-reference/domain-modifiers/ALIAS.md)"
		DomainModifierCaa        = "[`CAA`](../language-reference/domain-modifiers/CAA.md)"
		DomainModifierCsync      = "[`CSYNC`](../language-reference/domain-modifiers/CSYNC.md)"
		DomainModifierDhcid      = "[`DHCID`](../language-reference/domain-modifiers/DHCID.md)"
		DomainModifierDname      = "[`DNAME`](../language-reference/domain-modifiers/DNAME.md)"
		DomainModifierDnskey     = "[`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md)"
		FeatureAutoDNSSEC        = "[`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md)"
		DomainModifierDs         = "[`DS`](../language-reference/domain-modifiers/DS.md)"
		DomainModifierHTTPS      = "[`HTTPS`](../language-reference/domain-modifiers/HTTPS.md)"
		DomainModifierLoc        = "[`LOC`](../language-reference/domain-modifiers/LOC.md)"
		DomainModifierNaptr      = "[`NAPTR`](../language-reference/domain-modifiers/NAPTR.md)"
		DomainModifierOpenpgpkey = "[`OPENPGPKEY`](../language-reference/domain-modifiers/OPENPGPKEY.md)"
		DomainModifierPtr        = "[`PTR`](../language-reference/domain-modifiers/PTR.md)"
		DomainModifierRP         = "[`RP`](../language-reference/domain-modifiers/RP.md)"
		DomainModifierSMIMEA     = "[`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md)"
		DomainModifierSoa        = "[`SOA`](../language-reference/domain-modifiers/SOA.md)"
		DomainModifierSrv        = "[`SRV`](../language-reference/domain-modifiers/SRV.md)"
		DomainModifierSshfp      = "[`SSHFP`](../language-reference/domain-modifiers/SSHFP.md)"
		DomainModifierSvcb       = "[`SVCB`](../language-reference/domain-modifiers/SVCB.md)"
		DomainModifierTlsa       = "[`TLSA`](../language-reference/domain-modifiers/TLSA.md)"
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
			{ // record types, rendered as one combined list table
				Title:     "Record types",
				ListStyle: true,
				Features: []string{
					DomainModifierAlias,
					DomainModifierCaa,
					DomainModifierCsync,
					DomainModifierDhcid,
					DomainModifierDname,
					DomainModifierDnskey,
					DomainModifierDs,
					DomainModifierHTTPS,
					DomainModifierLoc,
					DomainModifierNaptr,
					DomainModifierOpenpgpkey,
					DomainModifierPtr,
					DomainModifierRP,
					DomainModifierSMIMEA,
					DomainModifierSoa,
					DomainModifierSrv,
					DomainModifierSshfp,
					DomainModifierSvcb,
					DomainModifierTlsa,
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
		setRecordType(DomainModifierAlias, "ALIAS")
		featureMap[FeatureAutoDNSSEC] = def.CanAutoDNSSEC
		setRecordType(DomainModifierCaa, "CAA")
		setRecordType(DomainModifierCsync, "CSYNC")
		setRecordType(DomainModifierDhcid, "DHCID")
		setRecordType(DomainModifierDname, "DNAME")
		setRecordType(DomainModifierDs, "DS")
		setRecordType(DomainModifierDnskey, "DNSKEY")
		setRecordType(DomainModifierHTTPS, "HTTPS")
		setRecordType(DomainModifierLoc, "LOC")
		setRecordType(DomainModifierNaptr, "NAPTR")
		setRecordType(DomainModifierOpenpgpkey, "OPENPGPKEY")
		setRecordType(DomainModifierPtr, "PTR")
		setRecordType(DomainModifierRP, "RP")
		setRecordType(DomainModifierSMIMEA, "SMIMEA")
		setRecordType(DomainModifierSoa, "SOA")
		setRecordType(DomainModifierSrv, "SRV")
		setRecordType(DomainModifierSshfp, "SSHFP")
		setRecordType(DomainModifierSvcb, "SVCB")
		setRecordType(DomainModifierTlsa, "TLSA")
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
	// FeatureTables renders in order. Tables with ListStyle render one row
	// per provider, listing the supported and unsupported features as
	// compact lists with counts; the rest render one emoji column per feature.
	FeatureTables []FeatureTableDef
}

// FeatureTableDef is one table of the provider feature matrix.
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
