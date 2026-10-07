package main

import (
	"cmp"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// readmeTableColumns is how many providers appear per row of the README table.
const readmeTableColumns = 5

// generateReadmeProvidersTable rewrites the "Supported Providers" section of
// README.md.
func generateReadmeProvidersTable() error {
	defs := publicDefinitions()
	slices.SortFunc(defs, func(a, b *providers.Definition) int {
		if order := cmp.Compare(strings.ToLower(a.FriendlyName), strings.ToLower(b.FriendlyName)); order != 0 {
			return order
		}
		return cmp.Compare(a.TypeName, b.TypeName)
	})

	cells := make([]string, 0, len(defs))
	for _, def := range defs {
		cells = append(cells, readmeTableCell(def))
	}

	// Pad the final row so every row has the same number of cells.
	for len(cells)%readmeTableColumns != 0 {
		cells = append(cells, "")
	}

	var content strings.Builder
	fmt.Fprintf(&content, "\nDNSControl supports %d DNS providers and registrars:\n\n", len(defs))

	content.WriteString(strings.Repeat("| ", readmeTableColumns))
	content.WriteString("|\n")

	content.WriteString("|")
	content.WriteString(strings.Repeat(" ----- |", readmeTableColumns))
	content.WriteString("\n")

	for i := 0; i < len(cells); i += readmeTableColumns {
		content.WriteString("| ")
		content.WriteString(strings.Join(cells[i:i+readmeTableColumns], " | "))
		content.WriteString(" |\n")
	}

	content.WriteString("\n")
	content.WriteString("¹also supports registrar functions\n")
	content.WriteString("²registrar only\n\n")

	replaceInlineContent(
		"README.md",
		"<!-- provider-table-start -->",
		"<!-- provider-table-end -->",
		content.String(),
	)

	return nil
}

// readmeTableCell renders one provider as a linked table cell, with a footnote
// marker describing which roles it can fill.
func readmeTableCell(def *providers.Definition) string {
	isDNSProvider := def.Kind.Has(providers.KindDNS)
	isRegistrar := def.Kind.Has(providers.KindRegistrar)

	footnote := ""
	switch {
	case isDNSProvider && isRegistrar:
		footnote = "¹"
	case isRegistrar:
		footnote = "²"
	}

	return fmt.Sprintf("[%s](%s)%s", def.FriendlyName, def.DocsURL, footnote)
}

// providerDocSlug uses the configured documentation path, independent of the
// canonical type and its aliases.
func providerDocSlug(def *providers.Definition) string {
	u, err := url.Parse(def.DocsURL)
	if err != nil || u.Path == "" {
		panic(fmt.Sprintf("provider %s has an invalid documentation URL: %q", def.TypeName, def.DocsURL))
	}
	return path.Base(u.Path)
}

func publicDefinitions() []*providers.Definition {
	var defs []*providers.Definition
	for _, def := range providers.AllDefinitions() {
		if def.TypeName != "NONE" {
			defs = append(defs, def)
		}
	}
	return defs
}
