package models

import (
	"fmt"
	"sync"

	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/qdm12/reprint"
	"golang.org/x/net/idna"
)

const (
	// DomainTag is the key used to store a copy of DomainConfig.Tag in the Metadata map.
	DomainTag = "dnscontrol_tag"

	// DomainUniqueName is the key used to store a copy of DomainConfig.UniqueName in the Metadata map.
	DomainUniqueName = "dnscontrol_uniquename"

	// DomainNameRaw is the key used to store a copy of DomainConfig.NameRaw in the Metadata map.
	DomainNameRaw = "dnscontrol_nameraw"

	// DomainNameUnicode is the key used to store a copy of DomainConfig.NameUnicode in the Metadata map.
	DomainNameUnicode = "dnscontrol_nameunicode"
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

func NewDomainConfig(name string) (*DomainConfig, error) {
	dcn := domaintags.MakeDomainNameVarieties(name) // TODO(tlim): Create a version of MakeDomainNameVarieties that returns an error on failure.
	//dcn := domaintags.DomainNameVarieties{NameASCII: name}
	dc := &DomainConfig{
		NameRaw:     dcn.NameRaw,
		Name:        dcn.NameASCII,
		NameUnicode: dcn.NameUnicode,
		Tag:         dcn.Tag,
		UniqueName:  dcn.UniqueName,
		DisplayName: dcn.DisplayName,

		// TODO(tlim): Eliminate the need for this Metedata. It was a hack to avoid changing legacy code but should be easier to eliminate now.
		Metadata: map[string]string{
			DomainUniqueName:  dcn.UniqueName,
			DomainNameRaw:     dcn.NameRaw,
			DomainNameUnicode: dcn.NameUnicode,
		},
	}

	return dc, nil
}

// FixLegacyDC calls .FixUp() on all records within DC.
func (dc *DomainConfig) FixLegacyDC() {
	dc.Records.FixLegacyRecords(dc.Name)
}

// FixLegacyDC calls .FixUp() on all records in recs.
func (recs Records) FixLegacyRecords(origin string) {
	for _, rec := range recs {
		rec.FixUp(origin)
	}
}

//func FixLegacyRecord(rec *models.RecordConfig, origin string) {
//	rec.FixUp(origin) // Hack. Populates .RDATA and .TypeNum if needed.
//}

// PostProcess performs and post-processing required after running dnsconfig.js and loading the result.
// It is called by dns.go's PostProcess() function.
func (dc *DomainConfig) PostProcess() {
	// Ensure the metadata map is initialized.
	if dc.Metadata == nil {
		dc.Metadata = map[string]string{}
	}

	//  HACK: Fill in names (This means models.NewDomainConfig() was not used.  We can eliminate this when legacy code is removed.
	ff := domaintags.MakeDomainNameVarieties(dc.Name) // FIXME: Slow.  Call only if needed.
	if dc.Tag == "" {
		dc.Tag = ff.Tag
	}
	if dc.NameRaw == "" {
		dc.NameRaw = ff.NameRaw
	}
	dc.Name = ff.NameASCII // We always overwrite this. It's a hack, but it works. Good enough until legacy code is removed.
	if dc.NameUnicode == "" {
		dc.NameUnicode = ff.NameUnicode
	}
	if dc.UniqueName == "" {
		dc.UniqueName = ff.UniqueName
	}
	if dc.DisplayName == "" {
		dc.DisplayName = ff.DisplayName
	}

	// Store the FixForms is Metadata so we don't have to change the signature of every function that might need them.
	// This is a bit ugly but avoids a huge refactor. Please avoid using these to make the future refactor easier.
	if dc.Tag != "" {
		dc.Metadata[DomainTag] = dc.Tag
	}
	dc.Metadata[DomainNameRaw] = dc.NameRaw
	dc.Metadata[DomainNameUnicode] = dc.NameUnicode
	dc.Metadata[DomainUniqueName] = dc.UniqueName
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
		if rec.IsModernType() {
			continue // Modern types handle punycode themselves.
		}

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
