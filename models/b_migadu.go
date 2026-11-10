package models

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"golang.org/x/net/idna"
)

func init() {
	RegisterBuilder("MIGADU_BUILDER", BuilderMigadu)
}

// The hosts, ports and labels Migadu fixes for every domain. Only the DKIM
// targets (which embed the mail domain) and the verification token differ
// between domains.
const (
	migaduMX1Target          = "aspmx1.migadu.com."
	migaduMX2Target          = "aspmx2.migadu.com."
	migaduDKIMTargetSuffix   = "._domainkey.migadu.com."
	migaduSPF                = "v=spf1 include:spf.migadu.com -all"
	migaduVerifyPrefix       = "hosted-email-verify="
	migaduAutoconfigTarget   = "autoconfig.migadu.com."
	migaduAutodiscoverTarget = "autodiscover.migadu.com."
	migaduSMTPTarget         = "smtp.migadu.com."
	migaduIMAPTarget         = "imap.migadu.com."
	migaduPOPTarget          = "pop.migadu.com."

	migaduMX1Priority = 10
	migaduMX2Priority = 20
	migaduDKIMKeys    = 3

	migaduAutoconfigLabel   = "autoconfig"
	migaduAutodiscoverLabel = "_autodiscover._tcp"
	migaduSubmissionsLabel  = "_submissions._tcp"
	migaduIMAPSLabel        = "_imaps._tcp"
	migaduPOP3SLabel        = "_pop3s._tcp"
)

// migaduOptionNames lists every option MIGADU_BUILDER accepts, sorted, so that
// a call with more than one bad option always reports the same one.
var migaduOptionNames = []string{
	"autoconfig",
	"autodiscover",
	"clientSRV",
	"dkim",
	"label",
	"mx",
	"spf",
	"verify",
	"wildcard",
}

// migaduOpts is the options object of MIGADU_BUILDER() after validation.
type migaduOpts struct {
	label  string
	verify string

	mx           bool
	wildcard     bool
	dkim         bool
	spf          bool
	autoconfig   bool
	autodiscover bool
	clientSRV    bool
}

// BuilderMigadu implements MIGADU_BUILDER(), which generates the DNS records a
// domain hosted at Migadu (https://www.migadu.com/) needs. The authoritative
// per-domain values are shown in the Migadu admin panel under
// Domains → <domain> → DNS.
//
// It emits provider facts only. SPF is opt-in, because a second SPF record at
// the same name is a PermError (RFC 7208 section 4.5), not a merge; DMARC is
// policy and is left to DMARC_BUILDER.
//
// args is [name, options]. builderOptions() (pkg/js/helpers.js) moves the
// options object into the argument list, so its values arrive as the JSON
// types dnsconfig.js produces: string, bool, float64 and map[string]any.
func BuilderMigadu(dc *DomainConfig, ttl uint32, args []any, subdomain string) (Records, error) {
	var name string
	if len(args) > 0 {
		if s, ok := args[0].(string); ok {
			name = s
		}
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New(`MIGADU_BUILDER: the first argument must be the mail domain name, for example MIGADU_BUILDER("example.com", { verify: "abcd1234" })`)
	}
	errPrefix := fmt.Sprintf("MIGADU_BUILDER(%q): ", name)

	if len(args) > 2 {
		return nil, fmt.Errorf("%sexpected the domain name and one options object, got %d arguments", errPrefix, len(args))
	}
	rawOpts := map[string]any{}
	if len(args) == 2 {
		o, ok := args[1].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%sthe second argument must be an options object, got %s", errPrefix, migaduJSONType(args[1]))
		}
		rawOpts = o
	}

	opts, err := migaduParseOptions(errPrefix, rawOpts)
	if err != nil {
		return nil, err
	}

	// The D_EXTEND() subdomain arrives as written in dnsconfig.js, not in IDNA
	// form. The records use the converted value, as non-builder records do.
	subdomainASCII := subdomain
	if subdomainASCII != "" {
		subdomainASCII, err = idna.ToASCII(subdomainASCII)
		if err != nil {
			return nil, fmt.Errorf("%sthe D_EXTEND() subdomain %q is not a valid IDNA name: %w", errPrefix, subdomain, err)
		}
		subdomainASCII = strings.ToLower(subdomainASCII)
	}

	label, err := dc.LabelFromDnsconfigjs(opts.label, subdomainASCII)
	if err != nil {
		return nil, fmt.Errorf("%s%w", errPrefix, err)
	}

	// The DKIM targets embed the mail domain, so it has to be the name the
	// records are created under. A mismatch would publish another domain's
	// DKIM targets.
	mailDomain, err := idna.ToASCII(strings.TrimRight(strings.TrimSpace(name), "."))
	if err != nil {
		return nil, fmt.Errorf("%sthe domain name is not a valid IDNA name: %w", errPrefix, err)
	}
	mailDomain = strings.ToLower(mailDomain)
	recordName := dc.Name
	if label != "@" {
		recordName = label + "." + dc.Name
	}
	if mailDomain != recordName {
		return nil, fmt.Errorf("%sthe mail domain %q is not the name these records are created under (%q); pass that name as the first argument, or set \"label\"", errPrefix, mailDomain, recordName)
	}

	var records Records
	add := func(recLabel string, typeNum uint16, rdata ...any) error {
		rec, err := dc.NewRecordConfig(recLabel, ttl, typeNum, rdata...)
		if err != nil {
			return fmt.Errorf("%scannot create the %s record at %q: %w", errPrefix, dnsutilv2.TypeToString(typeNum), recLabel, err)
		}
		rec.SubDomain = subdomainASCII
		records = append(records, rec)
		return nil
	}

	if opts.verify != "" {
		if err := add(label, dnsv2.TypeTXT, migaduVerifyPrefix+opts.verify); err != nil {
			return nil, err
		}
	}
	if opts.mx {
		mxLabels := []string{label}
		if opts.wildcard {
			mxLabels = append(mxLabels, migaduSubLabel("*", label))
		}
		for _, l := range mxLabels {
			if err := add(l, dnsv2.TypeMX, uint16(migaduMX1Priority), migaduMX1Target); err != nil {
				return nil, err
			}
			if err := add(l, dnsv2.TypeMX, uint16(migaduMX2Priority), migaduMX2Target); err != nil {
				return nil, err
			}
		}
	}
	if opts.dkim {
		for i := 1; i <= migaduDKIMKeys; i++ {
			key := fmt.Sprintf("key%d", i)
			if err := add(migaduSubLabel(key+"._domainkey", label), dnsv2.TypeCNAME, key+"."+mailDomain+migaduDKIMTargetSuffix); err != nil {
				return nil, err
			}
		}
	}
	if opts.spf {
		if err := add(label, dnsv2.TypeTXT, migaduSPF); err != nil {
			return nil, err
		}
	}
	if opts.autoconfig {
		if err := add(migaduSubLabel(migaduAutoconfigLabel, label), dnsv2.TypeCNAME, migaduAutoconfigTarget); err != nil {
			return nil, err
		}
	}
	if opts.autodiscover {
		if err := add(migaduSubLabel(migaduAutodiscoverLabel, label), dnsv2.TypeSRV, uint16(0), uint16(1), uint16(443), migaduAutodiscoverTarget); err != nil {
			return nil, err
		}
	}
	// RFC 6186 service discovery, so a client can find the submission and
	// retrieval endpoints from the address alone.
	if opts.clientSRV {
		for _, srv := range []struct {
			label  string
			port   uint16
			target string
		}{
			{migaduSubmissionsLabel, 465, migaduSMTPTarget},
			{migaduIMAPSLabel, 993, migaduIMAPTarget},
			{migaduPOP3SLabel, 995, migaduPOPTarget},
		} {
			if err := add(migaduSubLabel(srv.label, label), dnsv2.TypeSRV, uint16(0), uint16(1), srv.port, srv.target); err != nil {
				return nil, err
			}
		}
	}

	return records, nil
}

// migaduParseOptions validates the options object and fills in the defaults.
// rawOpts is only read, never written: one object may be shared between
// several D() blocks.
func migaduParseOptions(errPrefix string, rawOpts map[string]any) (migaduOpts, error) {
	opts := migaduOpts{
		label:        "@",
		mx:           true,
		dkim:         true,
		autoconfig:   true,
		autodiscover: true,
		clientSRV:    true,
	}

	names := slices.Sorted(maps.Keys(rawOpts))
	for _, name := range names {
		if !slices.Contains(migaduOptionNames, name) {
			return opts, fmt.Errorf("%sunknown option %q; valid options are %s", errPrefix, name, strings.Join(migaduOptionNames, ", "))
		}
	}

	for _, name := range names {
		value := rawOpts[name]
		var err error
		switch name {
		case "autoconfig":
			opts.autoconfig, err = migaduBool(errPrefix, name, value)
		case "autodiscover":
			opts.autodiscover, err = migaduBool(errPrefix, name, value)
		case "clientSRV":
			opts.clientSRV, err = migaduBool(errPrefix, name, value)
		case "dkim":
			opts.dkim, err = migaduBool(errPrefix, name, value)
		case "mx":
			opts.mx, err = migaduBool(errPrefix, name, value)
		case "spf":
			opts.spf, err = migaduBool(errPrefix, name, value)
		case "wildcard":
			opts.wildcard, err = migaduBool(errPrefix, name, value)

		case "label":
			opts.label, err = migaduString(errPrefix, name, value)

		case "verify":
			opts.verify, err = migaduString(errPrefix, name, value)
			if err == nil && strings.HasPrefix(opts.verify, migaduVerifyPrefix) {
				err = fmt.Errorf(`%soption %q must be the token only, without the %q prefix, for example "abcd1234"`, errPrefix, name, migaduVerifyPrefix)
			}
		}
		if err != nil {
			return opts, err
		}
	}

	if opts.wildcard && !opts.mx {
		return opts, fmt.Errorf(`%soption "wildcard" adds MX records, so it cannot be combined with "mx": false`, errPrefix)
	}

	return opts, nil
}

// The helpers below mirror their m365* counterparts in b_m365.go. They are
// kept separate so that this builder does not depend on another builder's
// internals.

// migaduBool reads a true/false option.
func migaduBool(errPrefix, name string, value any) (bool, error) {
	b, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%soption %q must be true or false", errPrefix, name)
	}
	return b, nil
}

// migaduString reads a string option and trims the whitespace around it.
func migaduString(errPrefix, name string, value any) (string, error) {
	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%soption %q must be a string", errPrefix, name)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("%soption %q must not be empty", errPrefix, name)
	}
	return s, nil
}

// migaduSubLabel places a record below the label of the mail domain.
func migaduSubLabel(recordPrefix, label string) string {
	if label == "@" {
		return recordPrefix
	}
	return recordPrefix + "." + label
}

// migaduJSONType names the type of a value the way it is written in
// dnsconfig.js, for use in error messages.
func migaduJSONType(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case []any:
		return "array"
	case nil:
		return "null"
	}
	return fmt.Sprintf("%T", value)
}
