package normalize

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nameutil"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/DNSControl/dnscontrol/v5/pkg/transform"
)

// Warning is a wrapper around error that can be used to indicate it should not
// stop execution, but is still likely a problem.
type Warning struct {
	error
}

// ValidateAndNormalizeConfig performs and normalization and/or validation of the IR.
func ValidateAndNormalizeConfig(config *models.DNSConfig) (errs []error) {
	if err := providers.Finalize(); err != nil {
		return []error{err}
	}
	err := processSplitHorizonDomains(config)
	if err != nil {
		return []error{err}
	}

	for _, domain := range config.Domains {
		var pTypes []string
		for _, provider := range domain.DNSProviderInstances {
			pTypes = append(pTypes, provider.ProviderType)
		}

		// Normalize Nameservers.
		for _, ns := range domain.Nameservers {
			// NB(tlim): Like any target, NAMESERVER() is input by the user
			// as a shortname or a FQDN+dot.
			if err := checkTarget(ns.Name); err != nil {
				errs = append(errs, err)
			}
			// Unlike any other FQDN in this system, it is stored as a FQDN without the trailing dot.
			n := nameutil.ToFqdnWithDot(ns.Name, domain.Name)
			ns.Name = strings.TrimSuffix(n, ".")
		}

		// Normalize Records.
		for _, rec := range domain.Records {
			if rec.TTL == 0 {
				rec.TTL = models.DefaultTTL
			}

			// Canonicalize Label:
			//if rec.GetLabel() == (domain.Name + ".") {
			//	// If label == ${domain}DOT, change to "@"
			//	rec.SetLabel("@", domain.Name)
			//} else if lab, suf := rec.GetLabel(), "."+domain.Name+"."; strings.HasSuffix(lab, suf) {
			//	// If label ends with DOT${domain}DOT, strip it to a short name.
			//	rec.SetLabel(lab[:len(lab)-len(suf)], domain.Name)
			//}
			// If label ends with dot, add to the list of errors.
			//if strings.HasSuffix(rec.GetLabel(), ".") {
			//	errs = append(errs, fmt.Errorf("label %q does not match D(%q)", rec.GetLabel(), domain.Name))
			//	return errs // Exit early.
			//}

			// IMPORT_TRANSFORM is a deferred command, not a provider record.
			// It is consumed below; the resulting records are validated afterwards.
			if rec.Type != "IMPORT_TRANSFORM" {
				if err := validateSupportedRecordTypes(rec, domain.Name, pTypes); err != nil {
					errs = append(errs, err)
				}
			}
			if err := checkLabel(rec.GetLabel(), rec.Type, domain.Name, rec.Metadata); err != nil {
				errs = append(errs, err)
			}

			if errs2 := checkTargets(rec, domain.Name); errs2 != nil {
				errs = append(errs, errs2...)
			}

			if errs2 := checkTLSASMIMEA(rec, domain.Name); errs2 != nil {
				errs = append(errs, errs2...)
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
				rd := rec.AsIMPORTTRANSFORM()
				transformTable := rd.TransformTable
				ttl := uint32(rd.TTL)
				suffixstrip := rd.SuffixStrip
				targetDomain := rd.TargetDomain
				table, err := transform.DecodeTransformTable(transformTable)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				c := config.FindDomain(targetDomain)
				if c == nil {
					err = fmt.Errorf("IMPORT_TRANSFORM mentions non-existent domain %q", targetDomain)
					errs = append(errs, err)
					continue
				}
				err = importTransform(c, domain, table, ttl, suffixstrip)
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
		identityFn := domainRecordIdentity(d)
		// Check that CNAMES don't have to co-exist with any other records
		errs = append(errs, checkCNAMEs(d, identityFn)...)
		// Check that only one SOA record exist for a zone
		errs = append(errs, checkMultipleSOAs(d)...)
		// Check that if any advanced record types are used in a domain, every provider for that domain supports them
		err := checkProviderCapabilities(d)
		if err != nil {
			errs = append(errs, err)
		}
		// Check for duplicates
		errs = append(errs, checkDuplicates(d.Records, identityFn)...)
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
