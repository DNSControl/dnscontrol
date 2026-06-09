package models

import (
	"fmt"
	"strings"

	"golang.org/x/net/idna"
)

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
	// TODO(tlim): Maybe add a debug mode that panics if name doesn't end with origin
	name = strings.ToLower(name)
	name = strings.TrimSuffix(name, dc.Name)
	if name == "" {
		return "@"
	}
	return strings.ToLower(name)
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
