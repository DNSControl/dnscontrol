package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

func generateGoreleaserFile() error {
	const file = ".goreleaser.yml"

	contentBytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	content := string(contentBytes)

	providerNames := allProviderNames()

	names := map[string]bool{}
	for _, name := range providerNames {
		names[strings.ToLower(name)] = true
	}

	// Aliases used in commit messages that differ from the registered provider name.
	for _, alias := range []string{"cloudflare", "hexonet", "gandi", "doh", "azuredns", "bunnydns"} {
		names[alias] = true
	}

	var lowered []string
	for name := range names {
		lowered = append(lowered, name)
	}
	sort.Strings(lowered)
	alternation := strings.Join(lowered, "|")
	newRegexp := fmt.Sprintf(`"(?i)((%s).*:)+.*"`, alternation)

	pattern := regexp.MustCompile(`"\(\?i\)\(\([a-z0-9_|]+\)\.\*:\)\+\.\*"`)
	if !pattern.MatchString(content) {
		return fmt.Errorf("could not find provider regexp in %s", file)
	}

	content = pattern.ReplaceAllLiteralString(content, newRegexp)

	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", file, err)
	}

	return nil
}
