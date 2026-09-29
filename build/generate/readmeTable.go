package main

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// readmeTableColumns is how many providers appear per row of the README table.
const readmeTableColumns = 5

// generateReadmeProvidersTable rewrites the "Supported Providers" section of
// README.md.
// name:  one passed to  RegisterDomainServiceProviderType/RegisterRegistrarType
// link: the same convention as updateProviderDocs()
// footnote markers: come from which registries the provider appears in.
func generateReadmeProvidersTable() error {
	names := allProviderNames()

	cells := make([]string, 0, len(names))
	for _, providerName := range names {
		cells = append(cells, readmeTableCell(providerName))
	}

	// Pad the final row so every row has the same number of cells.
	for len(cells)%readmeTableColumns != 0 {
		cells = append(cells, "")
	}

	var content strings.Builder
	fmt.Fprintf(&content, "\nDNSControl supports %d DNS providers and registrars:\n\n", len(names))

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
func readmeTableCell(providerName string) string {
	isDNSProvider := providers.DNSProviderTypes[providerName].Initializer != nil
	isRegistrar := providers.RegistrarTypes[providerName] != nil

	footnote := ""
	switch {
	case isDNSProvider && isRegistrar:
		footnote = "¹"
	case isRegistrar:
		footnote = "²"
	}

	return fmt.Sprintf("[`%s`](https://docs.dnscontrol.org/provider/%s)%s",
		providerName, providerDocSlug(providerName), footnote)
}

// providerDocSlug converts a provider name into the basename of its
// documentation page. This must match the naming used by updateProviderDocs().
func providerDocSlug(providerName string) string {
	return strings.ToLower(strings.ReplaceAll(providerName, "_", ""))
}
