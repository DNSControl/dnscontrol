package domaintags

import (
	"strings"

	"golang.org/x/net/idna"
)

// EfficientToASCII converts a domain name to its ASCII representation using
// IDNA, on error returns the original name, and avoids allocating new memory
// when possible. The final string is lowercased.
func EfficientToASCII(name string) string {
	nameIDN, err := idna.ToASCII(name)
	if err != nil {
		return name // Fallback to raw name on error.
	}
	nameIDN = strings.ToLower(nameIDN)

	// Avoid pointless duplication.
	if nameIDN == name {
		return name
	}
	return nameIDN
}
