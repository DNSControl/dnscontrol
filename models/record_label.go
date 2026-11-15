package models

import (
	"errors"
	"fmt"
	"net/netip"
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
	"CF_WORKER_ROUTE":               true, // Probably works either way. Test to find out.
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
		fmt.Printf("ERROR: LabelFromFQDNWithDot(%v) called WRONG. Maybe you want LabelFromShort?\n", name)
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
	fmt.Printf("ERROR: LabelFromFQDNWithDot(%v) called on WRONG domain (%q)\n", name, dc.Name)
	if newName == "" {
		return "@"
	}
	return newName
}

// isIPv6Label reports whether label is a bare IPv6 address, which is a
// permitted way to write a PTR label in an ip6.arpa zone. Parsing rather than
// looking for a colon keeps a label such as "foo:bar" out of this branch.
func isIPv6Label(label string) bool {
	addr, err := netip.ParseAddr(label)
	return err == nil && addr.Is6()
}

// LabelFromDnsconfigjs processes labels from `dnsconfig.js`.
//
// nameRaw can be any string that is acceptable in dnsconfig.js, including "@", "foo", "foo.bar", "foo.bar.baz.", etc.
//
// subdomainASCII must already be in IDNA (punycode, lowercase) form; the caller
// converts and memoizes it (see DNSConfig.ImportRawRecords), since the same
// subdomain is shared by every record in a D_EXTEND block.
//
// It enforces the "single dot" rule.
//
// It applies D_EXTEND() subdomain label rewriting.
//
// In a reverse zone, a label written with the zone name appended (as REV()
// returns it) has the zone name stripped.
//
// The returned label is relative to the zone (dc.Name).
func (dc *DomainConfig) LabelFromDnsconfigjs(rawLabel, subdomainASCII string) (string, error) {
	label, err := dc.labelFromDnsconfigjs(rawLabel, subdomainASCII)
	if err != nil {
		return "", err
	}
	return stripReverseZoneSuffix(label, dc.Name), nil
}

// stripReverseZoneSuffix removes ".zone" from the end of a label in a reverse
// zone (in-addr.arpa or ip6.arpa). REV("1.2.3.4") returns
// "4.3.2.1.in-addr.arpa" with no trailing dot, so such labels are accepted
// in reverse zones. In any other zone the label is returned unchanged.
func stripReverseZoneSuffix(label, zone string) string {
	if !isReverseZone(zone) {
		return label
	}
	return strings.TrimSuffix(label, "."+zone)
}

// labelFromDnsconfigjs implements LabelFromDnsconfigjs, except for the
// reverse zone suffix stripping.
func (dc *DomainConfig) labelFromDnsconfigjs(rawLabel, subdomainASCII string) (string, error) {
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
	case strings.HasSuffix(dc.Name, ".ip6.arpa") && isIPv6Label(labelASCII):
		// 2001:db8::1 sub -> 2001:db8::1 (leave it alone)
		return labelASCII, nil
	case strings.HasSuffix(labelASCII, ".in-addr.arpa"), strings.HasSuffix(labelASCII, ".ip6.arpa"):
		// 6.7.8.9.in-addr.arpa -> 6.7.8.9.in-addr.arpa (leave it alone;
		// LabelFromDnsconfigjs strips the zone suffix)
		//
		// NB(tlim): This used to strip the subdomain suffix. A label ending
		// in ".arpa" only ends with the subdomain when the subdomain is "a"
		// (an ip6.arpa nibble), and then the label was mangled.
		//if strings.HasSuffix(labelASCII, subdomainASCII) {
		//	return labelASCII[:len(labelASCII)-len(subdomainASCII)-1], nil
		//}
		return labelASCII, nil
	default:
		// one two -> one.two
		return labelASCII + "." + subdomainASCII, nil
	}
}

// labelFromDnsconfigjsHelper implements LabelFromDnsconfigjs's case when D_EXTEND() is not in play.
func (dc *DomainConfig) labelFromDnsconfigjsHelper(nameRaw string) (string, error) {

	name := nameRaw

	if name == "" {
		return "", errors.New(`label "" is invalid. Use "@" when a label is at the root (apex) of the zone`)
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
