package mustbe

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
)

// TargetHost returns a FQDN (or @) suitable as a target for CNAME and other records.
// * origin must be a FQDN without a trailing dot OR "". If "", no attempt is made to turn the string into a FQDN.
// * arg may be a string or it will be converted to a string.
// * Unicode is converted to PunyCode.
// * The result always ends with a "." unless it is "@".
// * It does not try to turn a FQDN into a shortname, but it will replace the origin with "@". The reason for not shortening it is that "preview" output is unclear when the user sees a shortname. Explicit is better than implicit.
// * This does not handle "*" (wildcards) since they are not valid in targets. That's why this is called TargetHost and not Host.
// Examples: (assume $origin = "domain.com")
// * `@` -> `@`
// * `foo.$origin.` -> `foo.$origin.`
// * `short` -> `short.$origin`
// * `other.com.` -> `other.com.`
// * NOT: `$origin.` -> `@`  (We no longer do this for the same reason we don't product shortnames any more.)
func TargetHost(origin string, arg any) string {
	if strings.HasSuffix(origin, ".") {
		panic("mustbe.Host must NOT be called with an origin ending with .")
	}

	var name string
	switch v := arg.(type) {
	case string:
		name = v
	case int:
		name = fmt.Sprintf("%d", arg)
	default:
		name = fmt.Sprintf("%v", arg)
	}

	// Special symbols:
	switch name {
	case "@":
		return name
	case "":
		return origin + "."
	}

	// Normalize it
	name = domaintags.EfficientToASCII(name)

	// // shorten origin to "@".
	// if origin != "" && name == origin+"." {
	//	return "@"
	//}

	// Is this already a FQDN? Return it.
	if strings.HasSuffix(name, ".") {
		return name
	}

	// origin not specified. Leave things as-is.
	if origin == "" {
		return name + "."

	}

	// This must be a shortname without a dot. Add origin and dot.
	return name + "." + origin + "."
}
