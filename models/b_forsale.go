package models

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"

	dnsv2 "codeberg.org/miekg/dns"
	"golang.org/x/net/idna"
)

func init() {
	RegisterBuilder("FORSALE_BUILDER", BuilderFORSALE)
}

// RFC 10023: the "_for-sale" TXT records signal that a registered domain name
// is available for purchase.
// https://www.rfc-editor.org/rfc/rfc10023
const (
	forSaleLabel   = "_for-sale"
	forSaleVersion = "v=FORSALE1;"

	// Each record is one character-string of at most 255 octets (Section
	// 2.4), which leaves 239 octets for the content value after the version
	// tag and a content tag such as "fcod=". The ABNF in Section 2.1 gives
	// the same limit.
	forSaleMaxValueLength = 255 - len(forSaleVersion) - len("fcod=")
)

// forSaleTags lists the content tags of Section 2.2, sorted. It is also the
// order in which the records are created.
var forSaleTags = []string{"fcod", "ftxt", "furi", "fval"}

// fval-value = fval-currency fval-amount (Section 2.1).
var forSaleValueRE = regexp.MustCompile(`^[A-Z]+[0-9]+(\.[0-9]+)?$`)

// BuilderFORSALE implements FORSALE_BUILDER(), which generates the "_for-sale"
// TXT records of RFC 10023 below a label.
//
// args is [label, options]. The options object is moved from the metadata list
// into the argument list by builderOptions() (pkg/js/helpers.js), so its
// values arrive as the JSON types dnsconfig.js produces.
//
// Without options a single "v=FORSALE1;" record is created, which the RFC
// allows. Otherwise there is one record per content value, because a record
// must not carry more than one tag-value pair.
func BuilderFORSALE(dc *DomainConfig, ttl uint32, args []any, subdomain string) (Records, error) {
	var label string
	if len(args) > 0 {
		if s, ok := args[0].(string); ok {
			label = strings.TrimSpace(s)
		}
	}
	if label == "" {
		return nil, errors.New(`FORSALE_BUILDER: the first argument must be the label of the domain name that is for sale, for example FORSALE_BUILDER("@", { furi: "https://example.com/buy" })`)
	}
	errPrefix := fmt.Sprintf("FORSALE_BUILDER(%q): ", label)

	if len(args) > 2 {
		return nil, fmt.Errorf("%sexpected the label and one options object, got %d arguments", errPrefix, len(args))
	}
	rawOpts := map[string]any{}
	if len(args) == 2 {
		o, ok := args[1].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%sthe second argument must be an options object, got %s", errPrefix, m365JSONType(args[1]))
		}
		rawOpts = o
	}

	// Section 2.6, Note 2: processors MUST ignore "_for-sale" names below
	// .arpa, so such a record could only ever be a mistake.
	if dc.Name == "arpa" || strings.HasSuffix(dc.Name, ".arpa") {
		return nil, fmt.Errorf("%sRFC 10023 does not apply to names below .arpa (%q)", errPrefix, dc.Name)
	}

	contents, err := forSaleParseOptions(errPrefix, rawOpts)
	if err != nil {
		return nil, err
	}

	// See BuilderM365: the D_EXTEND() subdomain arrives unconverted.
	subdomainASCII := subdomain
	if subdomainASCII != "" {
		subdomainASCII, err = idna.ToASCII(subdomainASCII)
		if err != nil {
			return nil, fmt.Errorf("%sthe D_EXTEND() subdomain %q is not a valid IDNA name: %w", errPrefix, subdomain, err)
		}
		subdomainASCII = strings.ToLower(subdomainASCII)
	}

	name, err := dc.LabelFromDnsconfigjs(label, subdomainASCII)
	if err != nil {
		return nil, fmt.Errorf("%s%w", errPrefix, err)
	}
	// Section 2.5: "_for-sale.*.example." is not a wildcard, so it does not
	// put every name below example. up for sale.
	if name == "*" || strings.HasPrefix(name, "*.") {
		return nil, fmt.Errorf("%sa wildcard cannot be put up for sale; RFC 10023 only allows \"_for-sale\" below a name that exists", errPrefix)
	}
	recordName := forSaleLabel
	if name != "@" {
		recordName = forSaleLabel + "." + name
	}

	if len(contents) == 0 {
		contents = []string{""}
	}
	records := make(Records, 0, len(contents))
	for _, content := range contents {
		rec, err := dc.NewRecordConfig(recordName, ttl, dnsv2.TypeTXT, forSaleVersion+content)
		if err != nil {
			return nil, fmt.Errorf("%scannot create the TXT record %q: %w", errPrefix, forSaleVersion+content, err)
		}
		rec.SubDomain = subdomainASCII
		records = append(records, rec)
	}
	return records, nil
}

// forSaleParseOptions validates the options object and returns the tag-value
// pairs ("furi=https://...") in the order the records are created: by tag,
// then in the order they were written.
func forSaleParseOptions(errPrefix string, rawOpts map[string]any) ([]string, error) {
	names := slices.Sorted(maps.Keys(rawOpts))
	for _, name := range names {
		if !slices.Contains(forSaleTags, name) {
			return nil, fmt.Errorf("%sunknown option %q; valid options are %s", errPrefix, name, strings.Join(forSaleTags, ", "))
		}
	}

	var contents []string
	for _, tag := range names {
		values, err := forSaleStrings(errPrefix, tag, rawOpts[tag])
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, value := range values {
			if err := forSaleCheckValue(errPrefix, tag, value); err != nil {
				return nil, err
			}
			// Section 2.1: every tag-value pair in the RRset MUST be unique.
			if seen[value] {
				return nil, fmt.Errorf("%soption %q lists %q twice; every tag-value pair must be unique", errPrefix, tag, value)
			}
			seen[value] = true
			contents = append(contents, tag+"="+value)
		}
	}
	return contents, nil
}

// forSaleStrings reads an option that is either a string or a list of strings.
func forSaleStrings(errPrefix, tag string, value any) ([]string, error) {
	switch v := value.(type) {
	case string:
		return []string{v}, nil
	case []any:
		if len(v) == 0 {
			return nil, fmt.Errorf("%soption %q must not be an empty list", errPrefix, tag)
		}
		values := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%soption %q must be a string or a list of strings, found %s in the list", errPrefix, tag, m365JSONType(item))
			}
			values = append(values, s)
		}
		return values, nil
	}
	return nil, fmt.Errorf("%soption %q must be a string or a list of strings, got %s", errPrefix, tag, m365JSONType(value))
}

// forSaleCheckValue checks one content value against Section 2.2. The values
// are used as written: leading or trailing spaces in a free text are part of
// the record, not something to clean up.
func forSaleCheckValue(errPrefix, tag, value string) error {
	if value == "" {
		return fmt.Errorf("%soption %q must not be empty", errPrefix, tag)
	}
	if len(value) > forSaleMaxValueLength {
		return fmt.Errorf("%soption %q is %d octets long; RFC 10023 allows at most %d, so that the record fits in one TXT string", errPrefix, tag, len(value), forSaleMaxValueLength)
	}

	switch tag {
	case "furi":
		// Section 2.2.3: exactly one URI (or IRI), percent-encoded where
		// RFC 3986 requires it, for example spaces as %20.
		if strings.ContainsFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
			return fmt.Errorf("%soption %q must be a single URI; percent-encode spaces as %%20: %q", errPrefix, tag, value)
		}
		u, err := url.Parse(value)
		if err != nil || u.Scheme == "" {
			return fmt.Errorf("%soption %q must be an absolute URI such as \"https://example.com/buy\", \"mailto:sales@example.com\" or \"tel:+1-201-555-0123\": %q", errPrefix, tag, value)
		}
	case "fval":
		// Section 2.2.4 and 3.3: an upper case currency code followed by
		// the amount, with an optional decimal point and no separators.
		if !forSaleValueRE.MatchString(value) {
			return fmt.Errorf("%soption %q must be an upper case currency code followed by the amount, for example \"EUR999\", \"USD750\" or \"BTC0.000010\": %q", errPrefix, tag, value)
		}
	}
	return nil
}
