package models

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/idna"
)

// subdomainExcludedTypes lists the record types whose labels are NOT rewritten
// when declared under a D_EXTEND() subdomain. This mirrors the exclusion list
// in the legacy recordBuilder (pkg/js/helpers.js).
var subdomainExcludedTypes = map[string]bool{
	"CLOUDFLAREAPI_SINGLE_REDIRECT": true,
	"CF_WORKER_ROUTE":               true,
	"ADGUARDHOME_A_PASSTHROUGH":     true,
	"ADGUARDHOME_AAAA_PASSTHROUGH":  true,
	"MIKROTIK_FWD":                  true,
	"MIKROTIK_NXDOMAIN":             true,
	"MIKROTIK_FORWARDER":            true,
}

// subdomainExcludedType reports whether typeName is excluded from D_EXTEND()
// subdomain label rewriting.
func subdomainExcludedType(typeName string) bool {
	return subdomainExcludedTypes[typeName]
}

var ipv4LabelRe = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)

// LabelFromDnsconfigjsSubdomain is like LabelFromDnsconfigjs but additionally
// applies D_EXTEND() subdomain label rewriting (mirroring the legacy
// recordBuilder logic in pkg/js/helpers.js). All string manipulation is done on
// post-IDNA (ASCII) strings: both the raw label and the subdomain are converted
// to ASCII first, then any concatenation is performed. The returned label is
// relative to the zone (dc.Name). If subdomain is empty, this is equivalent to
// LabelFromDnsconfigjs.
func (dc *DomainConfig) LabelFromDnsconfigjsSubdomain(rawLabel, subdomain string) (string, error) {
	if subdomain == "" {
		return dc.LabelFromDnsconfigjs(rawLabel)
	}

	// Convert the subdomain to ASCII (post-IDNA).
	subASCII, err := idna.ToASCII(subdomain)
	if err != nil {
		return "", fmt.Errorf("subdomain %q rejected by IDNA: %w", subdomain, err)
	}
	subASCII = strings.ToLower(subASCII)

	// Convert the label to ASCII (post-IDNA). "@" is preserved as-is.
	labelASCII := rawLabel
	if rawLabel != "@" {
		labelASCII, err = idna.ToASCII(rawLabel)
		if err != nil {
			return "", fmt.Errorf("label %q rejected by IDNA: %w", rawLabel, err)
		}
		labelASCII = strings.ToLower(labelASCII)
	}

	// All branches below operate on post-IDNA strings.
	switch {
	case labelASCII == "@":
		// @ sub -> sub
		return subASCII, nil
	case ipv4LabelRe.MatchString(labelASCII):
		// 1.2.3.4 sub -> 1.2.3.4 (leave it alone)
		return labelASCII, nil
	case strings.HasSuffix(dc.Name, ".ip6.arpa"):
		return subASCII, nil
	case strings.HasSuffix(labelASCII, ".in-addr.arpa"):
		// 4.3.2.1.in-addr.arpa -> 4.3 (strip the subdomain suffix)
		if strings.HasSuffix(labelASCII, subASCII) {
			return labelASCII[:len(labelASCII)-len(subASCII)-1], nil
		}
		return labelASCII, nil
	default:
		// one two -> one.two
		return labelASCII + "." + subASCII, nil
	}
}

// LabelFromShort takes a label and prepares it for use in a RecordConfig.
// name is a "shortname" ("foo", not "foo.example.com").
// name is assumed to be ASCII, not Unicode (which is what most APIs return).
// If name == "", "@" is returned.
func (dc *DomainConfig) LabelFromShort(name string) string {
	// TODO(tlim): Maybe add a debug mode that panics if name ends with "."?
	if name == "" {
		return "@"
	}
	return strings.ToLower(name)
}

// LabelFromFQDNNoDot takes a label and prepares it for use in a RecordConfig.
// Name is a FQDN without a dot ("foo.example.com").
// Name is assumed to be ASCII, not Unicode (which is what most APIs return).
// Name is assumed to end with the zone name (which is what most APIs return).
func (dc *DomainConfig) LabelFromFQDNNoDot(name string) string {
	if name == "" {
		return "@"
	}

	newName := strings.ToLower(name)

	if before, found := strings.CutSuffix(newName, "."+dc.Name); found {
		return before
	}
	if newName == dc.Name {
		return "@"
	}

	// These other possibilities all indicate the function was called wrong.
	fmt.Printf("DEBUG: LabelFromFQDNNoDot(%v) called\n", name)
	if newName == "" {
		return "@"
	}
	return newName
}

// LabelFromDnsconfigjs takes a label from dnsconfig.js and prepares it for use in a RecordConfig.
// This is where we implement the "if any dots, must be a FQDN" rule.
// Unicode is converted to ASCII via IDNA (PunyCode).
// An error is returned if this name is not in this zone.
// nameRaw can be an
// This does not check for stuttering. That should be done by the caller.
func (dc *DomainConfig) LabelFromDnsconfigjs(nameRaw string) (string, error) {

	// var name string
	// switch v := nameRaw.(type) {
	// case string:
	// 	name = v
	// // case float64:
	// // 	name = strconv.FormatInt(int64(v), 10)
	// default:
	// 	// name = fmt.Sprintf("%v", nameRaw)
	// 	panic(fmt.Sprintf("label %v is unknown type: %T", nameRaw, nameRaw))
	// }
	name := nameRaw

	if name == "" {
		return "", fmt.Errorf(`label "" is invalid. Use "@" when a label is at the root (apex) of the zone`)
	}
	if name == "@" {
		return name, nil
	}

	// Normalize to ASCII and Unicode
	nameASCII, err := idna.ToASCII(name)
	if err != nil {
		return "", fmt.Errorf("label %q rejected by IDNA: %w", name, err)
	}
	nameASCII = strings.ToLower(nameASCII)
	if nameASCII == name {
		nameASCII = name // re-use memory
	}

	// Strip the zone.
	if nameASCII == dc.Name+"." {
		return "@", nil
	}
	if before, found := strings.CutSuffix(nameASCII, "."+dc.Name+"."); found {
		return before, nil
	}

	if strings.HasSuffix(nameASCII, ".") {
		return "", fmt.Errorf("label %q is not in domain %q", name, dc.Name)
	}

	return nameASCII, nil
}
