package models

import (
	"fmt"
	"strings"
	"sync"

	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v4/pkg/nameutil"
	"github.com/qdm12/reprint"
	"golang.org/x/net/idna"
)

// DomainConfig describes a DNS domain (technically a DNS zone).
// Do not create your own `&models.DomainConfig{}`.  Use `models.NewDomainConfig(name)`.
type DomainConfig struct {
	NameRaw     string `json:"-"`    // name as entered by user in dnsconfig.js
	Name        string `json:"name"` // NO trailing "."   Converted to IDN (punycode) early in the pipeline.
	NameUnicode string `json:"-"`    // name in Unicode format

	Tag         string `json:"tag,omitempty"` // Split horizon tag.
	UniqueName  string `json:"uniquename"`    // .Name + "!" + .Tag (no !tag added if tag is "")
	DisplayName string `json:"-"`             // For TUI display: "canonical!tag" or "canonical!tag (unicode)"

	RegistrarName    string         `json:"registrar"`
	DNSProviderNames map[string]int `json:"dnsProviders"`

	Metadata         map[string]string `json:"meta,omitempty"`
	Records          Records           `json:"records"`
	Nameservers      []*Nameserver     `json:"nameservers,omitempty"`
	NameserversMutex sync.Mutex        `json:"-"`

	EnsureAbsent Records `json:"recordsabsent,omitempty"` // ENSURE_ABSENT
	KeepUnknown  bool    `json:"keepunknown,omitempty"`   // NO_PURGE

	Unmanaged       []*UnmanagedConfig `json:"unmanaged,omitempty"`                      // IGNORE()
	UnmanagedUnsafe bool               `json:"unmanaged_disable_safety_check,omitempty"` // DISABLE_IGNORE_SAFETY_CHECK

	IgnoreExternalDNS bool   `json:"ignore_external_dns,omitempty"` // IGNORE_EXTERNAL_DNS
	ExternalDNSPrefix string `json:"external_dns_prefix,omitempty"` // IGNORE_EXTERNAL_DNS prefix

	AutoDNSSEC string `json:"auto_dnssec,omitempty"` // "", "on", "off"
	// DNSSEC        bool              `json:"dnssec,omitempty"`

	// These fields contain instantiated provider instances once everything is linked up.
	// This linking is in two phases:
	// 1. Metadata (name/type) is available just from the dnsconfig. Validation can use that.
	// 2. Final driver instances are loaded after we load credentials. Any actual provider interaction requires that.
	RegistrarInstance    *RegistrarInstance     `json:"-"`
	DNSProviderInstances []*DNSProviderInstance `json:"-"`

	// Raw user-input from dnsconfig.js that will be processed into RecordConfigs later:
	RawRecords []RawRecordConfig `json:"rawrecords,omitempty"`

	// Pending work to do for each provider.  Provider may be a registrar or DSP.
	pendingCorrectionsMutex    sync.Mutex               // Protect pendingCorrections*
	pendingCorrections         map[string][]*Correction // Work to be done for each provider
	pendingCorrectionsOrder    []string                 // Call the providers in this order
	pendingActualChangeCount   map[string]int           // Number of changes to report (cumulative)
	pendingPopulateCorrections map[string][]*Correction // Corrections for zone creations at each provider
}

// NewDomainConfig creates and initializes a *models.DomainConfig.
func NewDomainConfig(name string) (*DomainConfig, error) {
	if strings.HasSuffix(name, ".") {
		return nil, fmt.Errorf("do not call NewDomainName with trailing dot: %q", name)
	}
	dc := &DomainConfig{
		Metadata: map[string]string{}, // Initialize so that nil checking is not required later.
	}
	dc.PopulateNamesFromRaw(name)

	return dc, nil
}

// MustNewDomainConfig is like NewDomainConfig but panics if initialization
// fails.  It is intended for use in variable initializations in unit tests.
func MustNewDomainConfig(name string) *DomainConfig {
	dc, err := NewDomainConfig(name)
	if err != nil {
		panic(err)
	}
	return dc
}

// ToFqdnWithDot converts a shortname to a FQDN+".".
// (Assume dc.Name == "bar.com")
// ToFqdnWithDot("foo")      = "foo.bar.com."   // Typical use.
// ToFqdnWithDot("@")        = "bar.com."       // Apex returns the apex.
// ToFqdnWithDot("")         = "bar.com."       // Apex returns the apex.
// ToFqdnWithDot("foo.com.") = "foo.com."       // FQDNs are unmodified.
// ToFqdnWithDot("foo"")     = "foo.bar.com."   // If origin ends with a ".", DTRT.
// Replaces dnsutilv1.AddOrigin().
// Similar to nameutil.ToFqdnWithDot() but uses the domain name from dc.
func (dc *DomainConfig) ToFqdnWithDot(s string) string {
	return nameutil.ToFqdnWithDot(s, dc.Name)
}

// ToFqdnNoDot is the same as ToFqdnWithDot but the result does not include a trailing ".".
// Replaces dnsutilv1.AddOrigin().
// Similar to DomainConfig.ToFqdnNoDot() but it takes origin from dc.Name.
func (dc *DomainConfig) ToFqdnNoDot(s string) string {
	return nameutil.ToFqdnNoDot(s, dc.Name)
}

// ToShort returns the shortname by stripping the domain's name from "name". If name is not below dc.Name, name is returned unchanged.
// If the name was shortened, it does not end with a ".". If the name was untouched, it ends with a ".".
// Calling ToShort on a string that is already a shortname is unsupported. Names that do not end with "." are assumed to be FQDNs without a trailing ".".
// Similar to name.ToShort() but uses the domain name from dc.
func (dc *DomainConfig) ToShort(name string) string {
	return nameutil.ToShort(name, dc.Name)
}

func (dc *DomainConfig) PopulateNamesFromRaw(rawname string) {
	dcn := domaintags.MakeDomainNameVarieties(rawname)
	dc.Name = dcn.NameASCII
	dc.Tag = dcn.Tag
	dc.NameRaw = dcn.NameRaw
	dc.NameUnicode = dcn.NameUnicode
	dc.DisplayName = dcn.DisplayName
	dc.UniqueName = dcn.UniqueName
}

// PostProcess performs and post-processing required after running dnsconfig.js and loading the result.
// It is called by dns.go's PostProcess() function.
func (dc *DomainConfig) PostProcess() {
}

// FixLegacyDC calls .FixUp() on all records within DC.
func (dc *DomainConfig) FixLegacyDC() {
	dc.Records.FixLegacyRecords(dc.Name)
}

// FixLegacyRecords calls .FixUp() on all records in recs.
func (recs Records) FixLegacyRecords(origin string) {
	for _, rec := range recs {
		rec.FixRD(origin)
	}
}

// GetSplitHorizonNames returns the domain's name, uniquename, and tag.
// Deprecated: use .Name, .Uniquename, and .Tag directly instead.
func (dc *DomainConfig) GetSplitHorizonNames() (name, uniquename, tag string) {
	return dc.Name, dc.UniqueName, dc.Tag
}

// GetUniqueName returns the domain's uniquename.
// Deprecated: dc.UniqueName directly instead.
func (dc *DomainConfig) GetUniqueName() (uniquename string) {
	return dc.UniqueName
}

// Copy returns a deep copy of the DomainConfig.
func (dc *DomainConfig) Copy() (*DomainConfig, error) {
	newDc := &DomainConfig{}
	err := reprint.FromTo(dc, newDc) // Deep copy
	return newDc, err
}

// Filter removes all records that don't match the filter f.
func (dc *DomainConfig) Filter(f func(r *RecordConfig) bool) {
	recs := []*RecordConfig{}
	for _, r := range dc.Records {
		if f(r) {
			recs = append(recs, r)
		}
	}
	dc.Records = recs
}

// Punycode will convert all records to punycode format.
// It will encode:
// - Name
// - NameFQDN
// - Target (CNAME and MX only).
// NOTE: This will go away when RCv3 is adopted.
func (dc *DomainConfig) Punycode() error {
	for _, rec := range dc.Records {
		// Update the label:
		t, err := idna.ToASCII(rec.GetLabelFQDN())
		if err != nil {
			return err
		}
		rec.SetLabelFromFQDN(t, dc.Name)

		// Set the target:
		switch rec.Type { // #rtype_variations
		case "ALIAS", "MX", "NS", "CNAME", "DNAME", "PTR", "SRV", "URL", "URL301", "FRAME", "R53_ALIAS", "AKAMAICDN", "AKAMAITLC", "CLOUDNS_WR", "PORKBUN_URLFWD", "BUNNY_DNS_RDR":
			// These rtypes are hostnames, therefore need to be converted (unlike, for example, an AAAA record)
			t, err := idna.ToASCII(rec.GetTargetField())
			if err != nil {
				return err
			}
			if err := rec.SetTarget(t); err != nil {
				return err
			}
		case "CLOUDFLAREAPI_SINGLE_REDIRECT", "CF_REDIRECT", "CF_TEMP_REDIRECT", "CF_WORKER_ROUTE", "ADGUARDHOME_A_PASSTHROUGH", "ADGUARDHOME_AAAA_PASSTHROUGH", "BUNNY_DNS_PZ", "MIKROTIK_FWD", "MIKROTIK_NXDOMAIN", "MIKROTIK_FORWARDER":
			if err := rec.SetTarget(rec.GetTargetField()); err != nil {
				return err
			}
		case "A", "AAAA", "CAA", "DHCID", "DNSKEY", "DS", "HTTPS", "LOC",
			"LUA", "NAPTR", "OPENPGPKEY", "RP", "SMIMEA", "SOA", "SSHFP", "SVCB",
			"TXT", "TLSA", "AZURE_ALIAS":
			// Nothing to do.
		default:
			return fmt.Errorf("Punycode rtype %v unimplemented", rec.Type)
		}
	}
	return nil
}

// StoreCorrections accumulates corrections in a thread-safe way.
func (dc *DomainConfig) StoreCorrections(providerName string, corrections []*Correction) {
	dc.pendingCorrectionsMutex.Lock()
	defer dc.pendingCorrectionsMutex.Unlock()

	if dc.pendingCorrections == nil {
		// First time storing anything.
		dc.pendingCorrections = make(map[string]([]*Correction))
		dc.pendingCorrections[providerName] = corrections
		dc.pendingCorrectionsOrder = []string{providerName}
	} else if c, ok := dc.pendingCorrections[providerName]; !ok {
		// First time key used
		dc.pendingCorrections[providerName] = corrections
		dc.pendingCorrectionsOrder = []string{providerName}
	} else {
		// Add to existing.
		dc.pendingCorrections[providerName] = append(c, corrections...)
		dc.pendingCorrectionsOrder = append(dc.pendingCorrectionsOrder, providerName)
	}
}

// GetCorrections returns the accumulated corrections for providerName.
func (dc *DomainConfig) GetCorrections(providerName string) []*Correction {
	dc.pendingCorrectionsMutex.Lock()
	defer dc.pendingCorrectionsMutex.Unlock()

	if dc.pendingCorrections == nil {
		// First time storing anything.
		return nil
	}
	if c, ok := dc.pendingCorrections[providerName]; ok {
		return c
	}
	return nil
}

// IncrementChangeCount accumulates change count in a thread-safe way.
func (dc *DomainConfig) IncrementChangeCount(providerName string, delta int) {
	dc.pendingCorrectionsMutex.Lock()
	defer dc.pendingCorrectionsMutex.Unlock()

	if dc.pendingActualChangeCount == nil {
		// First time storing anything.
		dc.pendingActualChangeCount = make(map[string](int))
	}
	dc.pendingActualChangeCount[providerName] += delta
}

// GetChangeCount accumulates change count in a thread-safe way.
func (dc *DomainConfig) GetChangeCount(providerName string) int {
	dc.pendingCorrectionsMutex.Lock()
	defer dc.pendingCorrectionsMutex.Unlock()

	return dc.pendingActualChangeCount[providerName]
}

// StorePopulateCorrections accumulates corrections in a thread-safe way.
func (dc *DomainConfig) StorePopulateCorrections(providerName string, corrections []*Correction) {
	dc.pendingCorrectionsMutex.Lock()
	defer dc.pendingCorrectionsMutex.Unlock()

	if dc.pendingPopulateCorrections == nil {
		dc.pendingPopulateCorrections = make(map[string][]*Correction, 1)
	}
	dc.pendingPopulateCorrections[providerName] = append(dc.pendingPopulateCorrections[providerName], corrections...)
}

// GetPopulateCorrections returns zone corrections in a thread-safe way.
func (dc *DomainConfig) GetPopulateCorrections(providerName string) []*Correction {
	dc.pendingCorrectionsMutex.Lock()
	defer dc.pendingCorrectionsMutex.Unlock()
	return dc.pendingPopulateCorrections[providerName]
}

// DomainNameVarieties returns the domain's names in various forms.
func (dc *DomainConfig) DomainNameVarieties() *domaintags.DomainNameVarieties {
	return &domaintags.DomainNameVarieties{
		NameRaw:     dc.NameRaw,
		NameASCII:   dc.Name,
		NameUnicode: dc.NameUnicode,
		UniqueName:  dc.UniqueName,
		Tag:         dc.Tag,
		HasBang:     dc.Tag != "",
	}
}
