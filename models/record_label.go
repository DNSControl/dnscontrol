package models

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/pkg/nameutil"
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

// SetLabelFromFQDN sets the .Name/.NameFQDN fields given a FQDN and origin.
// fqdn may have a trailing "." but it is not required.
// origin may not have a trailing dot.
// This is legacy code that will go away when everything uses models.NewRecordConfig*() functions.
func (rc *RecordConfig) SetLabelFromFQDN(fqdn, origin string) {
	// Assertions that make sure the function is being used correctly:
	if strings.HasSuffix(origin, ".") {
		panic(fmt.Errorf("origin (%s) is not supposed to end with a dot", origin))
	}
	if strings.HasSuffix(fqdn, "..") {
		panic(fmt.Errorf("fqdn (%s) is not supposed to end with double dots", origin))
	}

	// Trim off a trailing dot.
	fqdn = strings.TrimSuffix(fqdn, ".")

	fqdn = strings.ToLower(fqdn)
	origin = strings.ToLower(origin)
	rc.Name = nameutil.ToShort(fqdn, origin)
	rc.NameFQDN = fqdn
}

// subdomainExcludedType reports whether typeName is excluded from D_EXTEND()
// subdomain label rewriting.
func subdomainExcludedType(typeName string) bool {
	return subdomainExcludedTypes[typeName]
}

var ipv4LabelRe = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)

// LabelFromShort takes a label and prepares it for use in a RecordConfig.
// name is a "shortname" ("foo", not "foo.example.com").
// name is assumed to be ASCII, not Unicode (which is what most APIs return).
// If name == "", "@" is returned.
func (dc *DomainConfig) LabelFromShort(name string) string {
	if len(name) > 0 && name[len(name)-1] == '.' {
		fmt.Printf("ERROR: LabelFromShort(%v) called WRONG. Maybe you want LabelFromFQDNWithDot?\n", name)
	}

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
	if name == "@" {
		return name
	}
	if strings.HasSuffix(name, ".") {
		fmt.Printf("ERROR: LabelFromFQDNNoDot(%v) called WRONG. Maybe you want LabelFromFQDNWithDot?\n", name)
	}

	newName := strings.ToLower(name)

	if before, found := strings.CutSuffix(newName, "."+dc.Name); found {
		return before
	}
	if newName == dc.Name {
		return "@"
	}

	// These other possibilities all indicate the function was called wrong.
	fmt.Printf("ERROR: LabelFromFQDNNoDot(%v) called WRONG\n", name)
	if newName == "" {
		return "@"
	}
	return newName
}

// LabelFromFQDNWithDot takes a label and prepares it for use in a RecordConfig.
// Name is a FQDN with a dot ("foo.example.com.").
// Name is assumed to be ASCII, not Unicode (which is what most APIs return).
// Name is assumed to end with the zone name (which is what most APIs return).
func (dc *DomainConfig) LabelFromFQDNWithDot(name string) string {
	if name == "" {
		return "IMPOSSIBLE"
	}
	if !strings.HasSuffix(name, ".") {
		fmt.Printf("ERROR: LabelFromFQDNWithDot(%v) called WRONG. Maybe you want LabelFromFQDNNoDot?\n", name)
	}

	newName := strings.ToLower(name)

	if before, found := strings.CutSuffix(newName, "."+dc.Name+"."); found {
		return before
	}
	if newName == dc.Name+"." {
		return "@"
	}

	// These other possibilities all indicate the function was called wrong.
	fmt.Printf("ERROR: LabelFromFQDNWithDot(%v) called WRONG\n", name)
	if newName == "" {
		return "@"
	}
	return newName
}

// LabelFromDnsconfigjs is like LabelFromDnsconfigjs but additionally
// applies D_EXTEND() subdomain label rewriting (mirroring the legacy
// recordBuilder logic in pkg/js/helpers.js). All string manipulation is done on
// post-IDNA (ASCII) strings, then any concatenation is performed. The returned
// label is relative to the zone (dc.Name).
//
// subdomainASCII must already be in IDNA (punycode, lowercase) form; the caller
// converts and memoizes it (see DNSConfig.ImportRawRecords), since the same
// subdomain is shared by every record in a D_EXTEND block. If subdomainASCII is
// empty, this is equivalent to LabelFromDnsconfigjs.
func (dc *DomainConfig) LabelFromDnsconfigjs(rawLabel, subdomainASCII string) (string, error) {
	if subdomainASCII == "" {
		return dc.labelFromDnsconfigjsHelper(rawLabel)
	}

	// Convert the label to ASCII (post-IDNA). "@" is preserved as-is.
	labelASCII := rawLabel
	if rawLabel != "@" {
		var err error
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
		return subdomainASCII, nil
	case ipv4LabelRe.MatchString(labelASCII):
		// 1.2.3.4 sub -> 1.2.3.4 (leave it alone)
		return labelASCII, nil
	case strings.HasSuffix(dc.Name, ".ip6.arpa"):
		return subdomainASCII, nil
	case strings.HasSuffix(labelASCII, ".in-addr.arpa"):
		// 4.3.2.1.in-addr.arpa -> 4.3 (strip the subdomain suffix)
		if strings.HasSuffix(labelASCII, subdomainASCII) {
			return labelASCII[:len(labelASCII)-len(subdomainASCII)-1], nil
		}
		return labelASCII, nil
	default:
		// one two -> one.two
		return labelASCII + "." + subdomainASCII, nil
	}
}

// labelFromDnsconfigjsHelper takes a label from dnsconfig.js and prepares it for use in a RecordConfig.
//
// nameRaw can be any string that is acceptable in dnsconfig.js, including "@", "foo", "foo.bar", "foo.bar.baz", etc.
//
// - Unicode is converted to ASCII via IDNA (PunyCode).
// - An error is returned if this name is not in this zone.
//
// This does not check for stuttering. That should be done by the caller.
func (dc *DomainConfig) labelFromDnsconfigjsHelper(nameRaw string) (string, error) {

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
