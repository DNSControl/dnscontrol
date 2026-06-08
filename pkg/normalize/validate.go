package normalize

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/DNSControl/dnscontrol/v4/pkg/transform"
	dnsv1 "github.com/miekg/dns"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
)

// Returns false if target does not validate.
func checkIPv4(label string) error {
	if addr, err := netip.ParseAddr(label); err != nil || !addr.Is4() {
		return fmt.Errorf("WARNING: target (%v) is not an IPv4 address", label)
	}
	return nil
}

// Returns false if target does not validate.
func checkIPv6(label string) error {
	if addr, err := netip.ParseAddr(label); err != nil || !addr.Is6() {
		return fmt.Errorf("WARNING: target (%v) is not an IPv6 address", label)
	}
	return nil
}

// make sure target is valid reference for cnames, mx, etc.
func checkTarget(target string) error {
	if target == "@" {
		return nil
	}
	if target == "" {
		return errors.New("empty target (\"\"). Did you mean \"@\" instead?")
	}
	if strings.ContainsAny(target, `'" +,|!£$%&()=?^*ç°§;:<>[]()@`) {
		return fmt.Errorf("target (%v) includes invalid char", target)
	}
	if !strings.HasSuffix(target, ".in-addr.arpa.") && strings.Contains(target, "/") {
		return fmt.Errorf("target (%v) includes invalid char", target)
	}
	// If it contains a ".", it must end in a ".".
	if strings.ContainsRune(target, '.') && target[len(target)-1] != '.' {
		return fmt.Errorf("target (%v) must end with a (.) [https://docs.dnscontrol.org/language-reference/why-the-dot]", target)
	}
	return nil
}

// validateRecordTypes list of valid rec.Type values. Returns true if this is a real DNS record type, false means it is a pseudo-type used internally.
func validateRecordTypes(rec *models.RecordConfig, domain string, pTypes []string) error {
	if rec.IsModernType() {
		// Modern types do their own validation.
		return nil
	}
	switch rec.Type {
	// RCv3 records do not need this validation step.
	case "CLOUDFLAREAPI_SINGLE_REDIRECT", "RP", "DS":
		return nil
	}

	// #rtype_variations
	validTypes := map[string]bool{
		"A":                true,
		"AAAA":             true,
		"ALIAS":            false,
		"CAA":              true,
		"CNAME":            true,
		"DHCID":            true,
		"DNAME":            true,
		"DS":               true,
		"DNSKEY":           true,
		"HTTPS":            true,
		"IMPORT_TRANSFORM": false,
		"LOC":              true,
		"MX":               true,
		"NAPTR":            true,
		"NS":               true,
		"OPENPGPKEY":       true,
		"PTR":              true,
		"SMIMEA":           true,
		"SOA":              true,
		"SRV":              true,
		"SSHFP":            true,
		"SVCB":             true,
		"TLSA":             true,
		"TXT":              true,
	}
	_, ok := validTypes[rec.Type]
	if !ok {

		cType := providers.GetCustomRecordType(rec.Type)
		if cType == nil {
			//return fmt.Errorf("unsupported record type (%v) domain=%v name=%v", rec.Type, domain, rec.GetLabel())
			return fmt.Errorf("unsupported record type (%v) domain=%v name=%v Type=%s TypeNum=%d", rec.Type, domain, rec.GetLabel(), rec.Type, rec.TypeNum)
		}
		for _, providerType := range pTypes {
			if providerType != cType.Provider {
				return fmt.Errorf("custom record type %s is not compatible with provider type %s", rec.Type, providerType)
			}
		}
		// it is ok. Lets replace the type with real type and add metadata to say we checked it
		rec.Metadata["orig_custom_type"] = rec.Type
		if cType.RealType != "" {
			rec.Type = cType.RealType
		}
	}
	return nil
}

func errorRepeat(label, domain string) string {
	shortname := strings.TrimSuffix(label, "."+domain)
	return fmt.Sprintf(
		`The name "%s.%s." is an error (repeats the domain). Maybe instead of "%s" you intended "%s"? If not add DISABLE_REPEATED_DOMAIN_CHECK to this record to permit this as-is.`,
		label, domain,
		label,
		shortname,
	)
}

func checkLabel(label string, rType string, domain string, meta map[string]string) error {
	if label == "@" {
		return nil
	}
	if label == "" {
		return fmt.Errorf("empty %s label (\"\") in %s. Did you mean \"@\" instead?", rType, domain)
	}
	if label[len(label)-1] == '.' {
		return fmt.Errorf("label %s.%s ends with a (.)", label, domain)
	}
	if label == domain || strings.HasSuffix(label, "."+domain) {
		if m := meta["skip_fqdn_check"]; m != "true" {
			return errors.New(errorRepeat(label, domain))
		}
	}

	// Underscores are permitted in labels, but we print a warning unless they
	// are used in a way we consider typical.  Yes, we're opinionated here.

	// Don't warn for certain rtypes:
	if slices.Contains([]string{"SRV", "TLSA", "TXT", "LUA"}, rType) {
		return nil
	}
	// Don't warn for records that start with _
	// See https://github.com/DNSControl/dnscontrol/issues/829
	if strings.HasPrefix(label, "_") || strings.Contains(label, "._") || strings.HasPrefix(label, "sql-") {
		return nil
	}

	// Otherwise, warn.
	if strings.ContainsRune(label, '_') {
		return Warning{fmt.Errorf("label %s.%s contains \"_\" (can't be used in a URL)", label, domain)}
	}

	return nil
}

func checkSoa(expire uint32, minttl uint32, refresh uint32, retry uint32, mbox string) error {
	if expire <= 0 {
		return errors.New("SOA Expire must be > 0")
	}
	if minttl <= 0 {
		return errors.New("SOA Minimum TTL must be > 0")
	}
	if refresh <= 0 {
		return errors.New("SOA Refresh must be > 0")
	}
	if retry <= 0 {
		return errors.New("SOA Retry must be > 0")
	}
	if mbox == "" {
		return errors.New("SOA MBox must be specified")
	}
	if strings.ContainsRune(mbox, '@') {
		return errors.New("SOA MBox must have '.' instead of '@'")
	}
	return nil
}

// checkTargets returns true if rec.Target is valid for the rec.Type.
func checkTargets(rec *models.RecordConfig, domain string) (errs []error) {
	if rec.IsModernType() {
		// Modern types do their own validation.
		return nil
	}
	switch rec.Type {
	case "CLOUDFLAREAPI_SINGLE_REDIRECT", "RP", "DS":
		return nil
	}

	label := rec.GetLabel()
	target := rec.GetTargetField()
	check := func(e error) {
		if e != nil {
			err := fmt.Errorf("%s: %s %s.%s: %s", rec.FilePos, rec.Type, rec.GetLabel(), domain, e.Error())
			if _, ok := e.(Warning); ok {
				err = Warning{err}
			}
			errs = append(errs, err)
		}
	}
	switch rec.Type { // #rtype_variations
	case "A":
		check(checkIPv4(target))
	case "AAAA":
		check(checkIPv6(target))
	case "ALIAS":
		check(checkTarget(target))
	case "CNAME":
		check(checkTarget(target))
		if label == "@" {
			check(errors.New("cannot create CNAME record for bare domain"))
		}
		labelFQDN := dnsutilv1.AddOrigin(label, domain)
		targetFQDN := dnsutilv1.AddOrigin(target, domain)
		if labelFQDN == targetFQDN {
			check(errors.New("CNAME loop (target points at itself)"))
		}
	case "DNAME":
		check(checkTarget(target))
	case "LOC":
	case "MX":
		check(checkTarget(target))
	case "NAPTR":
		if target != "" {
			check(checkTarget(target))
		}
	case "NS":
		check(checkTarget(target))
		if label == "@" {
			check(errors.New("cannot create NS record for bare domain. Use NAMESERVER instead"))
		}
	case "PTR":
		check(checkTarget(target))
	case "SOA":
		check(checkSoa(rec.SoaExpire, rec.SoaMinttl, rec.SoaRefresh, rec.SoaRetry, rec.SoaMbox))
		check(checkTarget(target))
		if label != "@" {
			check(errors.New("SOA record is only valid for bare domain"))
		}
	case "SRV":
		check(checkTarget(target))
	case "LUA":
		upper := strings.ToUpper(rec.LuaRType)
		if upper == "" {
			check(errors.New("LUA records must specify an emitted rtype"))
			break
		}
		if _, ok := dnsv1.StringToType[upper]; !ok {
			check(fmt.Errorf("LUA emitted rtype (%s) is not a valid DNS type", rec.LuaRType))
		}
		rec.LuaRType = upper
	case "CAA", "DHCID", "DNSKEY", "DS", "HTTPS", "IMPORT_TRANSFORM", "OPENPGPKEY", "SMIMEA", "SSHFP", "SVCB", "TLSA", "TXT":
	default:
		if rec.Metadata["orig_custom_type"] != "" {
			// it is a valid custom type. We perform no validation on target
			return errs
		}
		errs = append(errs, fmt.Errorf("checkTargets: Unimplemented record type (%v) domain=%v name=%v",
			rec.Type, domain, rec.GetLabel()))
	}
	return errs
}

func transformCNAME(target, oldDomain, newDomain, suffixstrip string) string {
	// Canonicalize the target.  Add the newDomain minus the suffixstrip.
	//  foo -> foo.oldDomain.newDomain
	//  foo. -> foo.newDomain
	nd := strings.TrimPrefix(newDomain, suffixstrip+".")
	if strings.HasSuffix(target, ".") {
		return target + nd + "."
	}
	return dnsutilv1.AddOrigin(target, oldDomain) + "." + nd + "."
}

func newRec(rec *models.RecordConfig, ttl uint32) *models.RecordConfig {
	rec2, _ := rec.Copy()
	if ttl != 0 {
		rec2.TTL = ttl
	}
	return rec2
}

func transformLabel(label, suffixstrip string) (string, error) {
	if suffixstrip == "" {
		return label, nil
	}
	suffixstrip = "." + suffixstrip
	if !strings.HasSuffix(label, suffixstrip) {
		return "", fmt.Errorf("label %q does not end with %q", label, suffixstrip)
	}
	return label[:len(label)-len(suffixstrip)], nil
}

// import_transform imports the records of one zone into another, modifying records along the way.
func importTransform(srcDomain, dstDomain *models.DomainConfig,
	transforms []transform.IPConversion, ttl uint32, suffixstrip string,
) error {
	// Read srcDomain.Records, transform, and append to dstDomain.Records:
	// 1. Skip any that aren't A or CNAMEs.
	// 2. Append destDomainname to the end of the label.
	// 3. For CNAMEs, append destDomainname to the end of the target.
	// 4. For As, change the target as described the transforms.

	for _, rec := range srcDomain.Records {
		// If this record is marked to be skipped, skip it.
		if rec.Metadata["import_transform_skip"] != "" {
			continue
		}
		// If the dstDomain already has a record with this type+label, skip it.
		if dstDomain.Records.HasRecordTypeName(rec.Type, rec.GetLabelFQDN()) {
			continue
		}
		switch rec.Type {
		case "A":
			addr, _ := netip.ParseAddr(rec.GetTargetField())
			trs, err := transform.IPToList(addr, transforms)
			if err != nil {
				return fmt.Errorf("import_transform: TransformIP(%v, %v) returned err=%w", rec.GetTargetField(), transforms, err)
			}
			for _, tr := range trs {
				r := newRec(rec, ttl)
				l, err := transformLabel(r.GetLabelFQDN(), suffixstrip)
				if err != nil {
					return err
				}
				r.SetLabel(l, dstDomain.Name)
				if err := r.SetTarget(tr.String()); err != nil {
					return err
				}
				dstDomain.Records = append(dstDomain.Records, r)
			}
		case "CNAME":
			r := newRec(rec, ttl)
			l, err := transformLabel(r.GetLabelFQDN(), suffixstrip)
			if err != nil {
				return err
			}
			r.SetLabel(l, dstDomain.Name)
			if err := r.SetTarget(transformCNAME(r.GetTargetField(), srcDomain.Name, dstDomain.Name, suffixstrip)); err != nil {
				return err
			}
			dstDomain.Records = append(dstDomain.Records, r)
		default:
			// Anything else is ignored.
			continue
		}
	}
	return nil
}

// deleteImportTransformRecords deletes any IMPORT_TRANSFORM records from a domain.
func deleteImportTransformRecords(domain *models.DomainConfig) {
	for i := len(domain.Records) - 1; i >= 0; i-- {
		rec := domain.Records[i]
		if rec.Type == "IMPORT_TRANSFORM" {
			domain.Records = append(domain.Records[:i], domain.Records[i+1:]...)
		}
	}
}

// Warning is a wrapper around error that can be used to indicate it should not
// stop execution, but is still likely a problem.
type Warning struct {
	error
}

// ValidateAndNormalizeConfig performs and normalization and/or validation of the IR.
func ValidateAndNormalizeConfig(config *models.DNSConfig) (errs []error) {
	err := processSplitHorizonDomains(config)
	if err != nil {
		return []error{err}
	}

	for _, domain := range config.Domains {
		pTypes := []string{}
		for _, provider := range domain.DNSProviderInstances {
			pType := provider.ProviderType
			if pType == "-" {
				// "-" indicates that we don't yet know who the provider type
				// is.  This is probably due to the fact that `dnscontrol
				// check` doesn't read creds.json, which is where the TYPE is
				// set.  We will skip this test in this instance.  Later if
				// `dnscontrol preview` or `push` is used, the full check will
				// be performed.
				continue
			}
			//			// If NO_PURGE is in use, make sure this *isn't* a provider that *doesn't* support NO_PURGE.
			//			if domain.KeepUnknown && providers.ProviderHasCapability(pType, providers.CantUseNOPURGE) {
			//				errs = append(errs, fmt.Errorf("%s uses NO_PURGE which is not supported by %s(%s)", domain.Name, provider.Name, pType))
			//			}
		}

		// Normalize Nameservers.
		for _, ns := range domain.Nameservers {
			// NB(tlim): Like any target, NAMESERVER() is input by the user
			// as a shortname or a FQDN+dot.
			if err := checkTarget(ns.Name); err != nil {
				errs = append(errs, err)
			}
			// Unlike any other FQDN in this system, it is stored as a FQDN without the trailing dot.
			n := dnsutilv1.AddOrigin(ns.Name, domain.Name+".")
			ns.Name = strings.TrimSuffix(n, ".")
		}

		// Normalize Records.
		models.PostProcessRecords(domain.Records)
		// No need to call FixLegacyDC here. These records were created from dnsconfig.js, not from a provider.
		for _, rec := range domain.Records {
			if rec.TTL == 0 {
				rec.TTL = models.DefaultTTL
			}

			// Canonicalize Label:
			if rec.GetLabel() == (domain.Name + ".") {
				// If label == ${domain}DOT, change to "@"
				rec.SetLabel("@", domain.Name)
			} else if lab, suf := rec.GetLabel(), "."+domain.Name+"."; strings.HasSuffix(lab, suf) {
				// If label ends with DOT${domain}DOT, strip it to a short name.
				rec.SetLabel(lab[:len(lab)-len(suf)], domain.Name)
			}
			// If label ends with dot, add to the list of errors.
			if strings.HasSuffix(rec.GetLabel(), ".") {
				errs = append(errs, fmt.Errorf("label %q does not match D(%q)", rec.GetLabel(), domain.Name))
				return errs // Exit early.
			}

			// in-addr.arpa magic
			if strings.HasSuffix(domain.Name, ".in-addr.arpa") || strings.HasSuffix(domain.Name, ".ip6.arpa") {
				label := rec.GetLabel()
				if strings.HasSuffix(label, "."+domain.Name) {
					rec.SetLabel(label[0:(len(label)-len("."+domain.Name))], domain.Name)
				}
			}

			// Validate the unmodified inputs:
			if err := validateRecordTypes(rec, domain.Name, pTypes); err != nil {
				errs = append(errs, err)
			}
			if err := checkLabel(rec.GetLabel(), rec.Type, domain.Name, rec.Metadata); err != nil {
				errs = append(errs, err)
			}

			if errs2 := checkTargets(rec, domain.Name); errs2 != nil {
				errs = append(errs, errs2...)
			}

			// Canonicalize Targets.
			switch rec.Type { // #rtype_variations
			case "ALIAS", "CNAME", "MX", "NS", "SRV":
				// #rtype_variations
				// These record types have a target that is a hostname.
				// We normalize them to a FQDN so there is less variation to handle.  If a
				// provider API requires a shortname, the provider must do the shortening.
				origin := domain.Name + "."
				if rec.SubDomain != "" {
					origin = rec.SubDomain + "." + origin
				}
				if err := rec.SetTarget(dnsutilv1.AddOrigin(rec.GetTargetField(), origin)); err != nil {
					errs = append(errs, err)
				}
			case "A", "AAAA":
				if err := rec.SetTargetIP(rec.GetTargetIP()); err != nil {
					errs = append(errs, err)
				}
			case "PTR":
				var err error
				var name string
				if name, err = transform.PtrNameMagic(rec.GetLabel(), domain.Name); err != nil {
					errs = append(errs, err)
				}
				rec.SetLabel(name, domain.Name)
			case "CAA":
				// Per: https://www.iana.org/assignments/pkix-parameters/pkix-parameters.xhtml#caa-properties excluding reserved tags
				allowedTags := []string{"issue", "issuewild", "iodef", "contactemail", "contactphone", "issuemail", "issuevmc"}
				if !slices.Contains(allowedTags, rec.CaaTag) {
					errs = append(errs, fmt.Errorf("CAA tag %s is invalid", rec.CaaTag))
				}
			case "OPENPGPKEY":
				target := rec.GetTargetField()
				if target, err = transform.OPENPGPKEY(target); err != nil {
					errs = append(errs, err)
				} else {
					if err := rec.SetTarget(target); err != nil {
						errs = append(errs, err)
					}
				}
			case "TLSA":
				if rec.TlsaUsage > 3 {
					errs = append(errs, fmt.Errorf("TLSA Usage %d is invalid in record %s (domain %s)",
						rec.TlsaUsage, rec.GetLabel(), domain.Name))
				}
				if rec.TlsaSelector > 1 {
					errs = append(errs, fmt.Errorf("TLSA Selector %d is invalid in record %s (domain %s)",
						rec.TlsaSelector, rec.GetLabel(), domain.Name))
				}
				if rec.TlsaMatchingType > 2 {
					errs = append(errs, fmt.Errorf("TLSA MatchingType %d is invalid in record %s (domain %s)",
						rec.TlsaMatchingType, rec.GetLabel(), domain.Name))
				}
			case "SMIMEA":
				if rec.SmimeaUsage > 3 {
					errs = append(errs, fmt.Errorf("SMIMEA Usage %d is invalid in record %s (domain %s)",
						rec.SmimeaUsage, rec.GetLabel(), domain.Name))
				}
				if rec.SmimeaSelector > 1 {
					errs = append(errs, fmt.Errorf("SMIMEA Selector %d is invalid in record %s (domain %s)",
						rec.SmimeaSelector, rec.GetLabel(), domain.Name))
				}
				if rec.SmimeaMatchingType > 2 {
					errs = append(errs, fmt.Errorf("SMIMEA MatchingType %d is invalid in record %s (domain %s)",
						rec.SmimeaMatchingType, rec.GetLabel(), domain.Name))
				}
			}

			// Populate FQDN:
			rec.SetLabel(rec.GetLabel(), domain.Name)

			if _, ok := rec.Metadata["ignore_name_disable_safety_check"]; ok {
				errs = append(errs, errors.New("IGNORE_NAME_DISABLE_SAFETY_CHECK no longer supported. Please use DISABLE_IGNORE_SAFETY_CHECK for the entire domain"))
			}
		}
	}

	// SPF flattening
	if ers := flattenSPFs(config); len(ers) > 0 {
		errs = append(errs, ers...)
	}

	// Process IMPORT_TRANSFORM
	for _, domain := range config.Domains {
		for _, rec := range domain.Records {
			if rec.Type == "IMPORT_TRANSFORM" {
				suffixstrip := rec.Metadata["transform_suffixstrip"]
				table, err := transform.DecodeTransformTable(rec.Metadata["transform_table"])
				if err != nil {
					errs = append(errs, err)
					continue
				}
				c := config.FindDomain(rec.GetTargetField())
				if c == nil {
					err = fmt.Errorf("IMPORT_TRANSFORM mentions non-existent domain %q", rec.GetTargetField())
					errs = append(errs, err)
				}
				err = importTransform(c, domain, table, rec.TTL, suffixstrip)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	// Clean up:
	for _, domain := range config.Domains {
		deleteImportTransformRecords(domain)
	}
	// Run record transforms
	for _, domain := range config.Domains {
		if err := applyRecordTransforms(domain); err != nil {
			errs = append(errs, err)
		}
	}

	for _, d := range config.Domains {
		// Check that CNAMES don't have to co-exist with any other records
		errs = append(errs, checkCNAMEs(d)...)
		// Check that if any advanced record types are used in a domain, every provider for that domain supports them
		err := checkProviderCapabilities(d)
		if err != nil {
			errs = append(errs, err)
		}
		// Check for duplicates
		errs = append(errs, checkDuplicates(d.Records)...)
		// Check for different TTLs under the same label
		errs = append(errs, checkRecordSetHasMultipleTTLs(d.Records)...)
		// Check for inconsistent R53 weighted routing metadata within a group
		errs = append(errs, checkR53WeightedGroupConsistency(d.Records)...)
		// Validate FQDN consistency
		for _, r := range d.Records {
			if r.NameFQDN == "" || !strings.HasSuffix(r.NameFQDN, d.Name) {
				errs = append(errs, fmt.Errorf("record named '%s' does not have correct FQDN for domain '%s'. FQDN: %s", r.Name, d.Name, r.NameFQDN))
			}
		}
		// Verify AutoDNSSEC is valid.
		errs = append(errs, checkAutoDNSSEC(d)...)
	}

	// At this point we've munged anything that needs to be munged, and
	// validated anything that can be globally validated.
	// Let's ask the provider if there are any records they can't handle.
	for _, domain := range config.Domains { // For each domain..
		for _, provider := range domain.DNSProviderInstances { // For each provider...
			if provider.ProviderType == "-" {
				// "-" indicates that we don't yet know who the provider type
				// is.  This is probably due to the fact that `dnscontrol
				// check` doesn't read creds.json, which is where the TYPE is
				// set.  We will skip this test in this instance.  Later if
				// `dnscontrol preview` or `push` is used, the full check will
				// be performed.
				continue
			}
			if es := providers.AuditRecords(provider.ProviderType, domain.Records); len(es) != 0 {
				for _, e := range es {
					errs = append(errs, fmt.Errorf("%s rejects domain %s: %w", provider.ProviderType, domain.Name, e))
				}
			}
		}
	}

	return errs
}

// processSplitHorizonDomains finds "domain.tld!tag" domains and pre-processes them.
func processSplitHorizonDomains(config *models.DNSConfig) error {

	// Verify uniquenames are unique
	seen := map[string]bool{}
	for _, d := range config.Domains {
		uniquename := d.GetUniqueName()
		// empty tag == untagged ("example.com!" -> "example.com")
		uniquename = strings.TrimSuffix(uniquename, "!")
		if seen[uniquename] {
			return fmt.Errorf("duplicate domain name: %q", uniquename)
		}
		seen[uniquename] = true
	}

	return nil
}

func checkAutoDNSSEC(dc *models.DomainConfig) (errs []error) {
	if strings.ToLower(dc.RegistrarName) == "none" {
		return
	}
	if dc.AutoDNSSEC == "on" {
		for providerName := range dc.DNSProviderNames {
			if dc.RegistrarName != providerName {
				errs = append(errs, Warning{fmt.Errorf("AutoDNSSEC is enabled, but DNS provider %s does not match registrar %s", providerName, dc.RegistrarName)})
			}
		}
	}
	return
}

func checkCNAMEs(dc *models.DomainConfig) (errs []error) {
	cnames := map[string]bool{}
	proxiedCnames := map[string]bool{}
	for _, r := range dc.Records {
		if r.Type == "CNAME" {
			if cnames[r.GetLabel()] {
				errs = append(errs, fmt.Errorf("%s: cannot have multiple CNAMEs with same name: %s", r.FilePos, r.GetLabelFQDN()))
			}
			cnames[r.GetLabel()] = true
			if p, ok := r.Metadata["cloudflare_proxy"]; ok && (p == "on" || p == "full") {
				proxiedCnames[r.GetLabel()] = true
			}
		}
	}
	for _, r := range dc.Records {
		if cnames[r.GetLabel()] && r.Type != "CNAME" {
			// Allow AKAMAICDN and CNAME to have same name
			if r.Type == "AKAMAICDN" {
				continue
			}
			// Cloudflare proxied (flattened) CNAMEs are resolved internally
			// and never served as actual CNAME records, so the RFC 1034 §3.6.2
			// restriction does not apply.
			if proxiedCnames[r.GetLabel()] {
				continue
			}
			errs = append(errs, fmt.Errorf("%s: cannot have CNAME and %s record with same name: %s", r.FilePos, r.Type, r.GetLabelFQDN()))
		}
	}
	return
}

func checkDuplicates(records []*models.RecordConfig) (errs []error) {
	seen := map[string]*models.RecordConfig{}
	for _, r := range records {
		diffable := fmt.Sprintf("%s %s %s", r.GetLabelFQDN(), r.Type, r.ToComparableNoTTL())
		if seen[diffable] != nil {
			errs = append(errs, fmt.Errorf("exact duplicate record found: %s", diffable))
		}
		seen[diffable] = r
	}
	return errs
}

func checkRecordSetHasMultipleTTLs(records []*models.RecordConfig) (errs []error) {
	// The RFCs say that all records at a particular recordset should have
	// the same TTL.  Most providers don't care, and if they do the
	// dnscontrol provider code usually picks the lowest TTL for all of them.

	// General algorithm:
	// gather all records at a particular label.
	//     has[label] -> ttl -> type(s)
	// for each label, if there is more than one ttl, output ttl:A/TXT ttl:TXT/NS

	// Find the inconsistencies:
	m := make(map[string]map[uint32]map[string]bool)
	for _, r := range records {
		label := r.GetLabelFQDN()
		ttl := r.TTL
		rtype := r.Type

		if _, ok := m[label]; !ok {
			m[label] = make(map[uint32]map[string]bool)
		}
		if _, ok := m[label][ttl]; !ok {
			m[label][ttl] = make(map[string]bool)
		}
		m[label][ttl][rtype] = true
	}

	labels := make([]string, len(m))
	i := 0
	for k := range m {
		labels[i] = k
		i++
	}
	sort.Strings(labels)
	// NB(tlim): No need to de-dup labels. They come from map keys.

	for _, label := range labels {
		if len(m[label]) > 1 {
			// Invert for a more clear error message:
			r := make(map[string]map[uint32]bool)
			for ttl, rtypes := range m[label] {
				for rtype := range rtypes {
					if _, ok := r[rtype]; !ok {
						r[rtype] = make(map[uint32]bool)
					}
					r[rtype][ttl] = true
				}
			}

			// Report any cases where a RecordSet has > 1 different TTLs
			for rtype := range r {
				if len(r[rtype]) > 1 {
					result := formatInconsistency(r)
					errs = append(errs, Warning{fmt.Errorf("inconsistent TTLs at %q: %s", label, result)})
				}
			}
		}
	}

	return errs
}

func formatInconsistency(r map[string]map[uint32]bool) string {
	var rtypeResult []string
	for rtype, ttlsMap := range r {
		ttlList := make([]int, len(ttlsMap))
		i := 0
		for k := range ttlsMap {
			ttlList[i] = int(k)
			i++
		}

		sort.Ints(ttlList)

		rtypeResult = append(rtypeResult, fmt.Sprintf("%s:%v", rtype, commaSepInts(ttlList)))
	}
	sort.Strings(rtypeResult)
	return strings.Join(rtypeResult, " ")
}

func commaSepInts(list []int) string {
	slist := make([]string, len(list))
	for i, v := range list {
		slist[i] = strconv.Itoa(v)
	}
	return strings.Join(slist, ",")
}

// checkR53WeightedGroupConsistency validates that all records sharing the same
// label+type+set_identifier have identical weight and health_check_id, since
// they map to a single Route 53 ResourceRecordSet.
func checkR53WeightedGroupConsistency(records []*models.RecordConfig) (errs []error) {
	type groupMeta struct {
		weight      string
		healthCheck string
	}
	groups := map[string]groupMeta{}

	for _, rc := range records {
		sid := rc.Metadata["r53_set_identifier"]
		if sid == "" {
			continue
		}
		key := rc.GetLabelFQDN() + ":" + rc.Type + "!" + sid
		w := rc.Metadata["r53_weight"]
		hc := rc.Metadata["r53_health_check_id"]

		if existing, ok := groups[key]; ok {
			if existing.weight != w {
				errs = append(errs, fmt.Errorf("R53 weighted group %q at %s %s has inconsistent weights (%s vs %s)", sid, rc.Type, rc.GetLabelFQDN(), existing.weight, w))
			}
			if existing.healthCheck != hc {
				errs = append(errs, fmt.Errorf("R53 weighted group %q at %s %s has inconsistent health check IDs (%s vs %s)", sid, rc.Type, rc.GetLabelFQDN(), existing.healthCheck, hc))
			}
		} else {
			groups[key] = groupMeta{weight: w, healthCheck: hc}
		}
	}
	return errs
}

// We pull this out of checkProviderCapabilities() so that it's visible within
// the package elsewhere, so that our test suite can look at the list of
// capabilities we're checking and make sure that it's up-to-date.
var providerCapabilityChecks = []pairTypeCapability{
	// #rtype_variations
	// If a zone uses rType X, the provider must support capability Y.
	// {"X", providers.Y},
	capabilityCheck("AKAMAICDN", providers.CanUseAKAMAICDN),
	capabilityCheck("AKAMAITLC", providers.CanUseAKAMAITLC),
	capabilityCheck("ALIAS", providers.CanUseAlias),
	capabilityCheck("AUTODNSSEC", providers.CanAutoDNSSEC),
	capabilityCheck("AZURE_ALIAS", providers.CanUseAzureAlias),
	capabilityCheck("CAA", providers.CanUseCAA),
	capabilityCheck("DHCID", providers.CanUseDHCID),
	capabilityCheck("DNAME", providers.CanUseDNAME),
	capabilityCheck("DNSKEY", providers.CanUseDNSKEY),
	capabilityCheck("HTTPS", providers.CanUseHTTPS),
	capabilityCheck("LOC", providers.CanUseLOC),
	capabilityCheck("NAPTR", providers.CanUseNAPTR),
	capabilityCheck("OPENPGPKEY", providers.CanUseOPENPGPKEY),
	capabilityCheck("PTR", providers.CanUsePTR),
	capabilityCheck("R53_ALIAS", providers.CanUseRoute53Alias),
	capabilityCheck("RP", providers.CanUseRP),
	capabilityCheck("SMIMEA", providers.CanUseSMIMEA),
	capabilityCheck("SOA", providers.CanUseSOA),
	capabilityCheck("SRV", providers.CanUseSRV),
	capabilityCheck("SSHFP", providers.CanUseSSHFP),
	capabilityCheck("SVCB", providers.CanUseSVCB),
	capabilityCheck("TLSA", providers.CanUseTLSA),

	// DS needs special record-level checks
	{
		rType:     "DS",
		caps:      []providers.Capability{providers.CanUseDS, providers.CanUseDSForChildren},
		checkFunc: checkProviderDS,
	},
}

type pairTypeCapability struct {
	rType string
	// Capabilities the provider must implement if any records of type rType are found
	// in the zonefile. This is a disjunction - implementing at least one of the listed
	// capabilities is sufficient.
	caps []providers.Capability
	// checkFunc provides additional checks of each provider. This function should be
	// called if records of type rType are found in the zonefile.
	checkFunc func(pType string, _ models.Records) error
}

func capabilityCheck(rType string, caps ...providers.Capability) pairTypeCapability {
	return pairTypeCapability{
		rType: rType,
		caps:  caps,
	}
}

func providerHasAtLeastOneCapability(pType string, caps ...providers.Capability) bool {
	for _, cap := range caps {
		if providers.ProviderHasCapability(pType, cap) {
			return true
		}
	}

	return false
}

func checkProviderDS(pType string, records models.Records) error {
	switch {
	case providers.ProviderHasCapability(pType, providers.CanUseDS):
		// The provider can use DS records anywhere, including at the root
		return nil
	case !providers.ProviderHasCapability(pType, providers.CanUseDSForChildren):
		// Provider has no support for DS records
		return fmt.Errorf("provider %s uses DS records but does not support them", pType)
	default:
		// Provider supports DS records but not at the root
		for _, record := range records {
			if record.Type == "DS" && record.Name == "@" {
				return fmt.Errorf(
					"provider %s only supports child DS records, but zone had a record at the root (@)",
					pType,
				)
			}
		}
	}

	return nil
}

func checkProviderCapabilities(dc *models.DomainConfig) error {
	// Check if the zone uses a capability that the provider doesn't
	// support.
	for _, ty := range providerCapabilityChecks {
		hasAny := false
		switch ty.rType {
		case "AUTODNSSEC":
			if dc.AutoDNSSEC != "" {
				hasAny = true
			}
		default:
			for _, r := range dc.Records {
				if r.Type == ty.rType {
					hasAny = true
					break
				}
			}
		}
		if !hasAny {
			continue
		}
		for _, provider := range dc.DNSProviderInstances {
			if provider.ProviderType == "-" {
				// "-" indicates that we don't yet know who the provider type
				// is.  This is probably due to the fact that `dnscontrol
				// check` doesn't read creds.json, which is where the TYPE is
				// set.  We will skip this test in this instance.  Later if
				// `dnscontrol preview` or `push` is used, the full check will
				// be performed.
				continue
			}
			// fmt.Printf("  (checking if %q can %q for domain %q)\n", provider.ProviderType, ty.rType, dc.Name)
			if !providerHasAtLeastOneCapability(provider.ProviderType, ty.caps...) {
				return fmt.Errorf("domain %s uses %s records, but DNS provider type %s does not support them", dc.Name, ty.rType, provider.ProviderType)
			}

			if ty.checkFunc != nil {
				checkErr := ty.checkFunc(provider.ProviderType, dc.Records)
				if checkErr != nil {
					return fmt.Errorf("while checking %s records in domain %s: %w", ty.rType, dc.Name, checkErr)
				}
			}
		}
	}
	return nil
}

func applyRecordTransforms(domain *models.DomainConfig) error {
	for _, rec := range domain.Records {
		if rec.Type != "A" {
			continue
		}
		tt, ok := rec.Metadata["transform"]
		if !ok {
			continue
		}
		table, err := transform.DecodeTransformTable(tt)
		if err != nil {
			return err
		}
		ip := rec.GetTargetIP()
		newIPs, err := transform.IPToList(rec.GetTargetIP(), table)
		if err != nil {
			return err
		}
		for i, newIP := range newIPs {
			if i == 0 && newIP.Compare(ip) != 0 {
				// replace target of first record if different
				if err := rec.SetTarget(newIP.String()); err != nil {
					return err
				}
			} else if i > 0 {
				// any additional ips need identical records with the alternate ip added to the domain
				cpy, err := rec.Copy()
				if err != nil {
					return err
				}
				if err := cpy.SetTarget(newIP.String()); err != nil {
					return err
				}
				domain.Records = append(domain.Records, cpy)
			}
		}
	}
	return nil
}
