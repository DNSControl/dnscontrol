package main

import (
	"os"
	"path"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func generateOwnersFile() error {

	var ownersData strings.Builder
	for _, def := range providers.AllDefinitions() {
		providerMaintainer := def.Maintainer
		if providerMaintainer == "" {
			continue
		}
		if providerMaintainer == "NEEDS VOLUNTEER" {
			ownersData.WriteString("# ")
		}
		ownersData.WriteString("providers/")
		ownersData.WriteString(getProviderDirectory(def))
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

// Directory identity follows the implementation package, independent of type aliases.
func getProviderDirectory(def *providers.Definition) string {
	return path.Base(def.ImplementationType.Elem().PkgPath())
}
