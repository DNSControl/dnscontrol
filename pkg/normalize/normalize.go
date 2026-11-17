package normalize

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
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

	// Phase 1: Apply. Run the features that rewrite or add records.
	errs = append(errs, flattenSPFs(config)...)
	errs = append(errs, applyImportTransforms(config)...)
	for _, domain := range config.Domains {
		if err := applyRecordTransforms(domain); err != nil {
			errs = append(errs, err)
		}
	}

	// Phase 2: Normalize. Put nameservers into canonical form. Records need
	// no work here: they are put into canonical form when they are constructed.
	for _, domain := range config.Domains {
		errs = append(errs, normalizeNameservers(domain)...)
	}

	// Phase 3: Check each record, including those added in phase 1.
	for _, domain := range config.Domains {
		errs = append(errs, checkRecords(domain)...)
	}

	// Phase 4: Check each zone as a whole.
	for _, domain := range config.Domains {
		errs = append(errs, checkZone(domain)...)
	}

	// Phase 5: Audit. At this point we've munged anything that needs to be
	// munged, and validated anything that can be globally validated.
	// Let's ask the provider if there are any records they can't handle.
	for _, domain := range config.Domains {
		errs = append(errs, auditRecords(domain)...)
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

// checkRecords runs the per-record checks on every record in a domain.
func checkRecords(domain *models.DomainConfig) (errs []error) {
	var pTypes []string
	for _, provider := range domain.DNSProviderInstances {
		pTypes = append(pTypes, provider.ProviderType)
	}

	for _, rec := range domain.Records {
		if err := validateSupportedRecordTypes(rec, domain.Name, pTypes); err != nil {
			errs = append(errs, err)
		}
		if err := checkLabel(rec.GetLabel(), rec.Type, domain.Name, rec.Metadata); err != nil {
			errs = append(errs, err)
		}
		errs = append(errs, checkTargets(rec, domain.Name)...)
		if err := checkObsoleteMetadata(rec); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// checkZone runs the zone-wide checks on a domain.
func checkZone(d *models.DomainConfig) (errs []error) {
	identityFn := domainRecordIdentity(d)
	// Check that CNAMES don't have to co-exist with any other records
	errs = append(errs, checkCNAMEs(d, identityFn)...)
	// Check that only one SOA record exist for a zone
	errs = append(errs, checkMultipleSOAs(d)...)
	// Check that every provider for the domain supports the domain-level features it uses
	if err := checkProviderCapabilities(d); err != nil {
		errs = append(errs, err)
	}
	// Check for duplicates
	errs = append(errs, checkDuplicates(d.Records, identityFn)...)
	// Check for different TTLs under the same label
	errs = append(errs, checkRecordSetHasMultipleTTLs(d.Records)...)
	// Check for inconsistent R53 weighted routing metadata within a group
	errs = append(errs, checkR53WeightedGroupConsistency(d.Records)...)
	// Validate FQDN consistency
	errs = append(errs, checkFQDNs(d)...)
	// Verify AutoDNSSEC is valid.
	errs = append(errs, checkAutoDNSSEC(d)...)
	return errs
}
