package models

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"golang.org/x/net/idna"
)

func init() {
	RegisterBuilder("M365_BUILDER", BuilderM365)
}

// The targets and labels Microsoft documents for the worldwide cloud. Every
// target has a matching <record>Target option that replaces it; that is how the
// sovereign clouds (GCC High, DoD, 21Vianet), DANE-enabled tenants and the DKIM
// format Microsoft introduced in May 2025 are configured.
const (
	m365MXTargetSuffix        = ".mail.protection.outlook.com."
	m365AutodiscoverTarget    = "autodiscover.outlook.com."
	m365SIPFederationTarget   = "sipfed.online.lync.com."
	m365MDMEnrollmentTarget   = "enterpriseenrollment-s.manage.microsoft.com."
	m365MDMRegistrationTarget = "enterpriseregistration.windows.net."

	m365AutodiscoverLabel    = "autodiscover"
	m365Selector1Label       = "selector1._domainkey"
	m365Selector2Label       = "selector2._domainkey"
	m365SIPFederationLabel   = "_sipfederationtls._tcp"
	m365MDMEnrollmentLabel   = "enterpriseenrollment"
	m365MDMRegistrationLabel = "enterpriseregistration"

	m365SIPFederationPriority = 100
	m365SIPFederationWeight   = 1
	m365SIPFederationPort     = 5061
)

// m365OptionNames lists every option M365_BUILDER accepts, sorted. Options are
// validated in this order so that a call with more than one bad option always
// reports the same one (Go maps have no order).
var m365OptionNames = []string{
	"autodiscover",
	"autodiscoverTarget",
	"dkim",
	"dkimSelector1Target",
	"dkimSelector2Target",
	"domainGUID",
	"initialDomain",
	"label",
	"mdm",
	"mdmEnrollmentTarget",
	"mdmRegistrationTarget",
	"mx",
	"mxPriority",
	"mxTarget",
	"sipFederationTarget",
	"skypeForBusiness",
	"verificationToken",
}

// m365Opts is the options object of M365_BUILDER() after validation. Every
// target is fully qualified (it ends with a "."); initialDomain is not, because
// it is embedded in the middle of the DKIM targets.
type m365Opts struct {
	label             string
	verificationToken string
	domainGUID        string
	initialDomain     string

	mx           bool
	autodiscover bool
	dkim         bool

	skypeForBusiness bool
	mdm              bool

	mxPriority            uint16
	mxTarget              string
	autodiscoverTarget    string
	dkimSelector1Target   string
	dkimSelector2Target   string
	sipFederationTarget   string
	mdmEnrollmentTarget   string
	mdmRegistrationTarget string
}

// BuilderM365 implements M365_BUILDER(), which generates the DNS records
// Microsoft 365 requires for one domain. The records and the targets are taken
// from Microsoft's documentation:
// https://learn.microsoft.com/en-us/microsoft-365/enterprise/external-domain-name-system-records
// The pages for the individual records are listed under "References" in
// documentation/language-reference/domain-modifiers/M365_BUILDER.md.
//
// args is [name, options]. The options object is moved from the metadata list
// into the argument list by m365Options() (pkg/js/helpers.js), so its values
// arrive as the JSON types dnsconfig.js produces: string, bool, float64 and
// map[string]any. Anything else is a user error, never a reason to panic.
//
// Errors quote the call itself because builder errors are reported without a
// file position (see DNSConfig.ImportRawRecords).
func BuilderM365(dc *DomainConfig, ttl uint32, args []any, subdomain string) (Records, error) {
	var name string
	if len(args) > 0 {
		if s, ok := args[0].(string); ok {
			name = s
		}
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New(`M365_BUILDER: the first argument must be the Microsoft 365 domain name, for example M365_BUILDER("example.com", { initialDomain: "contoso.onmicrosoft.com" })`)
	}
	errPrefix := fmt.Sprintf("M365_BUILDER(%q): ", name)

	if len(args) > 2 {
		return nil, fmt.Errorf("%sexpected the domain name and one options object, got %d arguments", errPrefix, len(args))
	}
	rawOpts := map[string]any{}
	if len(args) == 2 {
		o, ok := args[1].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%sthe second argument must be an options object, got %s", errPrefix, m365JSONType(args[1]))
		}
		rawOpts = o
	}

	opts, err := m365ParseOptions(errPrefix, rawOpts)
	if err != nil {
		return nil, err
	}

	// The builder path receives the D_EXTEND() subdomain unconverted, so it is
	// converted here. Everything downstream (the label and rec.SubDomain) uses
	// the result, which keeps the records identical to the non-builder path.
	subdomainASCII := subdomain
	if subdomainASCII != "" {
		subdomainASCII, err = idna.ToASCII(subdomainASCII)
		if err != nil {
			return nil, fmt.Errorf("%sthe D_EXTEND() subdomain %q is not a valid IDNA name: %w", errPrefix, subdomain, err)
		}
		subdomainASCII = strings.ToLower(subdomainASCII)
	}

	// The label of the Microsoft 365 domain within the zone. Every other label
	// is this one plus a fixed prefix; LabelFromDnsconfigjs() must not be
	// applied again, or the D_EXTEND() subdomain would be appended twice.
	label, err := dc.LabelFromDnsconfigjs(opts.label, subdomainASCII)
	if err != nil {
		return nil, fmt.Errorf("%s%w", errPrefix, err)
	}

	// The domain token is only derived when a record actually needs it. A
	// domain that publishes exact targets (or only the MDM records) never needs
	// one, and must not be rejected for a name Microsoft would not accept.
	domainGUID := opts.domainGUID
	if domainGUID == "" && ((opts.mx && opts.mxTarget == "") || (opts.dkim && opts.dkimSelector1Target == "")) {
		domainGUID, err = m365DomainGUID(dc, errPrefix, name, label)
		if err != nil {
			return nil, err
		}
	}

	selector1Target, selector2Target := opts.dkimSelector1Target, opts.dkimSelector2Target
	if opts.dkim && selector1Target == "" {
		if opts.initialDomain == "" {
			return nil, fmt.Errorf(`%soption "initialDomain" is required to derive the DKIM targets ("dkim" defaults to true); set it to your tenant's initial domain, for example "contoso.onmicrosoft.com", set "dkimSelector1Target" and "dkimSelector2Target" to the values shown in the Microsoft 365 admin center, or set "dkim": false`, errPrefix)
		}
		selector1Target = "selector1-" + domainGUID + "._domainkey." + opts.initialDomain + "."
		selector2Target = "selector2-" + domainGUID + "._domainkey." + opts.initialDomain + "."
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

	if opts.verificationToken != "" {
		if err := add(label, dnsv2.TypeTXT, opts.verificationToken); err != nil {
			return nil, err
		}
	}
	if opts.mx {
		mxTarget := opts.mxTarget
		if mxTarget == "" {
			mxTarget = domainGUID + m365MXTargetSuffix
		}
		if err := add(label, dnsv2.TypeMX, opts.mxPriority, mxTarget); err != nil {
			return nil, err
		}
	}
	if opts.autodiscover {
		if err := add(m365SubLabel(m365AutodiscoverLabel, label), dnsv2.TypeCNAME, opts.autodiscoverTarget); err != nil {
			return nil, err
		}
	}
	if opts.dkim {
		if err := add(m365SubLabel(m365Selector1Label, label), dnsv2.TypeCNAME, selector1Target); err != nil {
			return nil, err
		}
		if err := add(m365SubLabel(m365Selector2Label, label), dnsv2.TypeCNAME, selector2Target); err != nil {
			return nil, err
		}
	}
	if opts.skypeForBusiness {
		if err := add(m365SubLabel(m365SIPFederationLabel, label), dnsv2.TypeSRV,
			uint16(m365SIPFederationPriority), uint16(m365SIPFederationWeight), uint16(m365SIPFederationPort),
			opts.sipFederationTarget); err != nil {
			return nil, err
		}
	}
	if opts.mdm {
		if err := add(m365SubLabel(m365MDMRegistrationLabel, label), dnsv2.TypeCNAME, opts.mdmRegistrationTarget); err != nil {
			return nil, err
		}
		if err := add(m365SubLabel(m365MDMEnrollmentLabel, label), dnsv2.TypeCNAME, opts.mdmEnrollmentTarget); err != nil {
			return nil, err
		}
	}

	return records, nil
}

// m365ParseOptions validates the options object and fills in the defaults.
// Nothing from rawOpts is retained: the caller frees the args after the builder
// returns (see DNSConfig.ImportRawRecords).
func m365ParseOptions(errPrefix string, rawOpts map[string]any) (m365Opts, error) {
	opts := m365Opts{
		label:                 "@",
		mx:                    true,
		autodiscover:          true,
		dkim:                  true,
		autodiscoverTarget:    m365AutodiscoverTarget,
		sipFederationTarget:   m365SIPFederationTarget,
		mdmEnrollmentTarget:   m365MDMEnrollmentTarget,
		mdmRegistrationTarget: m365MDMRegistrationTarget,
	}

	names := slices.Sorted(maps.Keys(rawOpts))
	for _, name := range names {
		if !slices.Contains(m365OptionNames, name) {
			return opts, fmt.Errorf("%sunknown option %q; valid options are %s", errPrefix, name, strings.Join(m365OptionNames, ", "))
		}
	}

	for _, name := range names {
		value := rawOpts[name]
		var err error
		switch name {
		case "autodiscover":
			opts.autodiscover, err = m365Bool(errPrefix, name, value)
		case "dkim":
			opts.dkim, err = m365Bool(errPrefix, name, value)
		case "mdm":
			opts.mdm, err = m365Bool(errPrefix, name, value)
		case "mx":
			opts.mx, err = m365Bool(errPrefix, name, value)
		case "skypeForBusiness":
			opts.skypeForBusiness, err = m365Bool(errPrefix, name, value)

		case "autodiscoverTarget":
			opts.autodiscoverTarget, err = m365Target(errPrefix, name, "autodiscover.office365.us", value)
		case "dkimSelector1Target":
			opts.dkimSelector1Target, err = m365Target(errPrefix, name, "selector1-contoso-com._domainkey.contoso.n-v1.dkim.mail.microsoft", value)
		case "dkimSelector2Target":
			opts.dkimSelector2Target, err = m365Target(errPrefix, name, "selector2-contoso-com._domainkey.contoso.n-v1.dkim.mail.microsoft", value)
		case "mdmEnrollmentTarget":
			opts.mdmEnrollmentTarget, err = m365Target(errPrefix, name, "enterpriseenrollment-s.manage.microsoft.us", value)
		case "mdmRegistrationTarget":
			opts.mdmRegistrationTarget, err = m365Target(errPrefix, name, "enterpriseregistration.windows.net", value)
		case "mxTarget":
			opts.mxTarget, err = m365Target(errPrefix, name, "contoso.mail.protection.office365.us", value)
		case "sipFederationTarget":
			opts.sipFederationTarget, err = m365Target(errPrefix, name, "sipfed.online.gov.skypeforbusiness.us", value)

		case "initialDomain":
			// The initial domain is the middle part of the derived DKIM
			// targets, where the trailing dot belongs at the very end.
			var target string
			target, err = m365Target(errPrefix, name, "contoso.onmicrosoft.com", value)
			opts.initialDomain = strings.TrimSuffix(target, ".")

		case "label":
			opts.label, err = m365String(errPrefix, name, value)

		case "domainGUID":
			var guid string
			guid, err = m365String(errPrefix, name, value)
			if err == nil && strings.Contains(guid, ".") {
				err = fmt.Errorf(`%soption %q must be a single label without dots, for example "example-com"`, errPrefix, name)
			}
			opts.domainGUID = guid

		case "verificationToken":
			var token string
			token, err = m365String(errPrefix, name, value)
			if err == nil && (!strings.HasPrefix(token, "MS=") || token == "MS=") {
				err = fmt.Errorf(`%soption %q must be the full TXT value shown in the Microsoft 365 admin center, for example "MS=ms12345678"`, errPrefix, name)
			}
			opts.verificationToken = token

		case "mxPriority":
			opts.mxPriority, err = m365Priority(errPrefix, name, value)
		}
		if err != nil {
			return opts, err
		}
	}

	// Only while the records are created: a selector left behind after DKIM was
	// switched off describes no record and must not block the call.
	if opts.dkim && (opts.dkimSelector1Target == "") != (opts.dkimSelector2Target == "") {
		return opts, fmt.Errorf(`%soptions "dkimSelector1Target" and "dkimSelector2Target" must be set together; Microsoft publishes a value for both selectors`, errPrefix)
	}

	return opts, nil
}

// m365Bool reads a true/false option.
func m365Bool(errPrefix, name string, value any) (bool, error) {
	b, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%soption %q must be true or false", errPrefix, name)
	}
	return b, nil
}

// m365String reads a string option and trims the whitespace around it.
func m365String(errPrefix, name string, value any) (string, error) {
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

// m365Target reads a host name option and returns it fully qualified.
//
// The trailing dot is not cosmetic: mustbe.TargetHost() appends the origin to
// any target that does not have one, so a value copied from the Microsoft 365
// admin center would silently become "contoso.mail.protection.office365.us.example.com.".
func m365Target(errPrefix, name, example string, value any) (string, error) {
	s, err := m365String(errPrefix, name, value)
	if err != nil {
		return "", err
	}
	// Exactly one trailing dot, whether the value was copied with one or not.
	s = strings.TrimRight(s, ".")
	// The dot has to be an inner one: ".com" is not a host name either.
	if i := strings.Index(s, "."); i <= 0 {
		return "", fmt.Errorf("%soption %q must be a fully qualified host name, for example %q", errPrefix, name, example)
	}
	return s + ".", nil
}

// m365Priority reads the MX preference. Numbers from dnsconfig.js arrive as
// float64, including the ones the user wrote as integers.
func m365Priority(errPrefix, name string, value any) (uint16, error) {
	f, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("%soption %q must be a number", errPrefix, name)
	}
	if f != math.Trunc(f) || f < 0 || f > math.MaxUint16 {
		return 0, fmt.Errorf("%soption %q must be a whole number between 0 and 65535", errPrefix, name)
	}
	return uint16(f), nil
}

// m365DomainGUID derives the token Microsoft assigns to a domain. Microsoft
// replaces every dot of the domain name with a dash, but only for names that
// contain no dash of their own.
//
// The derivation is only correct if the domain name is the name the records are
// created under, so that is checked rather than assumed: the name in the call
// and the place the records end up are set independently, and a mismatch would
// otherwise publish the MX and DKIM records of a different domain.
func m365DomainGUID(dc *DomainConfig, errPrefix, name, label string) (string, error) {
	m365Domain, err := idna.ToASCII(strings.TrimRight(strings.TrimSpace(name), "."))
	if err != nil {
		return "", fmt.Errorf("%sthe domain name is not a valid IDNA name: %w", errPrefix, err)
	}
	m365Domain = strings.ToLower(m365Domain)

	recordName := dc.Name
	if label != "@" {
		recordName = label + "." + dc.Name
	}
	if m365Domain != recordName {
		return "", fmt.Errorf("%sthe Microsoft 365 domain %q is not the name these records are created under (%q); pass that name as the first argument, set \"label\", or set \"domainGUID\"", errPrefix, m365Domain, recordName)
	}

	if strings.Contains(m365Domain, "-") {
		return "", fmt.Errorf(`%s"domainGUID" has no default for a domain name that contains a dash, because Microsoft assigns an opaque value (for example "myexample-com01c"); copy the value from the MX record shown in the Microsoft 365 admin center`, errPrefix)
	}

	return strings.ReplaceAll(m365Domain, ".", "-"), nil
}

// m365SubLabel places a record below the label of the Microsoft 365 domain.
func m365SubLabel(recordPrefix, label string) string {
	if label == "@" {
		return recordPrefix
	}
	return recordPrefix + "." + label
}

// m365JSONType names the type of a value the way it is written in
// dnsconfig.js, for use in error messages.
func m365JSONType(value any) string {
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
