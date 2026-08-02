package mustbe

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

// TargetHost returns a FQDN (or @) suitable as a target for CNAME and other records.
// What is a target?
// * A "target" is a hostname used in fields of a DNS record. For example, in "foo IN MX 10 bar.example.com.", "bar" is the target.
// * A target is stored as a FQDN+"." with all Unicode converted to ASCII (PunyCode).
// * It is never stored as "@" or a shortname.
// * If you require a shortname, use dc.TargetAsShort(target) or dc.TargetAsShortOrAt(target).
//
// * origin must be a FQDN without a trailing dot. Violations cause a panic so that this is caught before code is shipped.
// * if origin is "", a special mode is activated that disables most action.
//   - This is for legacy situations where the origin is unknown.
//   - This will eventually be an error.
//   - In this mode, Either the original string is returned, or if that string is "", "UNKNOWNORIGIN." is returned. This is the only situation where this function might return "@".
//
// * Normal operation:
//   - arg may be a string or it will be converted to a string using fmt.Printf("%v").
//   - Unicode is converted to PunyCode.
//   - The result always ends with a "."
//   - The reason for not storing the short or "@" version is that "preview" output becomes ambiguous. Explicit is better than implicit.
//   - Wildcards ("@") are rejected since they are not valid in targets. That's why this is called TargetHost and not Host.
//
// * origin == ".":
//   - This is a special flag for TargetIsFqdnNoDot mode.
//
// * Examples: (assume $origin = "domain.com")
//   - `@` -> $origin
//   - `foo.$origin.` -> `foo.$origin.`
//   - `$origin.` -> `$origin.`
//   - `other.com.` -> `other.com.`
//   - `short` -> `short.$origin`
func TargetHost(origin string, isEnabled nrc.Flags, arg any) string {
	// Check for programmer error. Origin shouldn't end in ".", unless it == "." which is a special mode.
	if origin != "." && strings.HasSuffix(origin, ".") {
		panic(fmt.Sprintf("mustbe.Host must NOT be called with an origin ending with . (origin=%q)", origin))
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

	// Special mode for legacy situations.
	if origin == "" {
		if name == "" {
			return "UNKNOWNORIGIN."
		}
		return name
	}

	// TODO(tlim):
	// origin == "." is a special flag that means the same thing as
	// isEnabled.TargetIsFqdnNoDot == true.
	// I'm not sure why we have two ways to signal this mode. Too much late-night
	// coding, I suspect.
	// Or, perhaps this was written before I decided to change the signatures of
	// the Make*() functions to include isEnabled, after which I should have
	// removed the origin=="." code.
	// In the future we should eliminate the origin=".".

	if origin == "." && name == "" {
		return "."
	}

	// Special symbols:
	switch name {
	case "@", "":
		return origin + "."
	}

	// Normalize it
	name = domaintags.EfficientToASCII(name)

	if isEnabled.TargetIsFqdnNoDot {
		// The dot is unexpected but we'll allow it.
		if name[len(name)-1] == '.' {
			return name
		}
		// Add the dot.
		return name + "."
	}

	// Is this already a FQDN? Return it.
	if strings.HasSuffix(name, ".") {
		return name
	}

	// This must be a shortname without a dot. Add origin and dot.
	return name + "." + origin + "."
}

// TargetHostSRV is like TargetHost with the exception that "." and "" have special meaning and are left alone.
func TargetHostSRV(origin string, isEnabled nrc.Flags, arg any) string {

	var name string
	switch v := arg.(type) {
	case string:
		name = v
	case int:
		name = fmt.Sprintf("%d", arg)
	default:
		name = fmt.Sprintf("%v", arg)
	}

	if name == "" || name == "." {
		return name
	}

	return TargetHost(origin, isEnabled, name)
}
