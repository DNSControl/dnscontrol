package main

import (
	"os"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func generateLabelerFile() error {

	var labelerData strings.Builder
	for _, def := range providers.AllDefinitions() {
		if def.Maintainer == "" {
			continue
		}
		providerDirectory := getProviderDirectory(def)
		labelerData.WriteString("provider-")
		labelerData.WriteString(def.TypeName)
		labelerData.WriteString(":\n")
		labelerData.WriteString("  - changed-files:\n")
		labelerData.WriteString("      - any-glob-to-any-file: providers/")
		labelerData.WriteString(providerDirectory)
		labelerData.WriteString("/**\n")
	}

	return os.WriteFile(".github/labeler.yml", []byte(labelerData.String()), 0o644)
}
