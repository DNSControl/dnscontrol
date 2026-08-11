package main

import (
	"os"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

func generateLabelerFile() error {
	maintainers := providers.ProviderMaintainers
	sortedProviderNames := getSortedProviderNames(maintainers)

	var labelerData strings.Builder
	for _, providerName := range sortedProviderNames {
		providerDirectory := getProviderDirectory(providerName)
		labelerData.WriteString("provider-")
		labelerData.WriteString(providerName)
		labelerData.WriteString(":\n")
		labelerData.WriteString("  - changed-files:\n")
		labelerData.WriteString("      - any-glob-to-any-file: providers/")
		labelerData.WriteString(providerDirectory)
		labelerData.WriteString("/**\n")
	}

	return os.WriteFile(".github/labeler.yml", []byte(labelerData.String()), 0o644)
}
