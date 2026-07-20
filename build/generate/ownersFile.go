package main

import (
	"os"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

func generateOwnersFile() error {
	maintainers := providers.ProviderMaintainers
	sortedProviderNames := getSortedProviderNames(maintainers)

	var ownersData strings.Builder
	for _, providerName := range sortedProviderNames {
		providerMaintainer := maintainers[providerName]
		if providerMaintainer == "NEEDS VOLUNTEER" {
			ownersData.WriteString("# ")
		}
		ownersData.WriteString("providers/")
		ownersData.WriteString(getProviderDirectory(providerName))
		ownersData.WriteString(" ")
		ownersData.WriteString(providerMaintainer)
		ownersData.WriteString("\n")
	}

	// Overall maintainer
	ownersData.WriteString("\n")
	ownersData.WriteString("* @TomOnTime\n")
	ownersData.WriteString(".goreleaser.yml @cafferata @TomOnTime\n")
	ownersData.WriteString("documentation/ @cafferata @TomOnTime\n")

	file, err := os.Create(".github/CODEOWNERS")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(ownersData.String())
	if err != nil {
		return err
	}

	return nil
}

func getProviderDirectory(providerName string) string {
	// Strip the underscores from the providerName constants
	providerDirectory := strings.ToLower(
		strings.ReplaceAll(
			providerName, "_", "",
		),
	)

	// These providers use a different directory name
	if providerDirectory == "cloudflareapi" {
		providerDirectory = "cloudflare"
	}
	if providerDirectory == "dnsoverhttps" {
		providerDirectory = "doh"
	}

	return providerDirectory
}

func getSortedProviderNames(maintainers map[string]string) []string {
	providerNameSorted := make([]string, 0, len(maintainers))
	for providerNameKey := range maintainers {
		providerNameSorted = append(providerNameSorted, providerNameKey)
	}
	sort.Strings(providerNameSorted)

	return providerNameSorted
}
