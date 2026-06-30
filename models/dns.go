package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"golang.org/x/net/idna"

	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

// DefaultTTL is applied to any DNS record without an explicit TTL.
const DefaultTTL = uint32(300)

// DNSConfig describes the desired DNS configuration, usually loaded from dnsconfig.js.
type DNSConfig struct {
	Registrars         []*RegistrarConfig            `json:"registrars"`
	DNSProviders       []*DNSProviderConfig          `json:"dns_providers"`
	Domains            []*DomainConfig               `json:"domains"`
	RegistrarsByName   map[string]*RegistrarConfig   `json:"-"`
	DNSProvidersByName map[string]*DNSProviderConfig `json:"-"`
	SkipRecordAudit    bool                          `json:"skiprecordaudit,omitempty"`
}

// ImportRawRecords iterates over the dc.RawRecords from each domain,
// converting each to a RecordConfig and deleting the raw version (to save
// memory). This is how records get from dnsconfig.js to dc.Records.
func (config *DNSConfig) ImportRawRecords() error {
	// subdomainToASCII converts a D_EXTEND() subdomain to its IDNA (punycode)
	// form. idna.ToASCII is comparatively expensive and every record in a
	// D_EXTEND block shares the same subdomain value, so the results are
	// memoized (keyed by the raw subdomain) across the whole import.
	subdomainCache := map[string]string{}
	subdomainToASCII := func(s string) (string, error) {
		if s == "" {
			return "", nil
		}
		if a, ok := subdomainCache[s]; ok {
			return a, nil
		}
		a, err := idna.ToASCII(s)
		if err != nil {
			return "", err
		}
		a = strings.ToLower(a)
		subdomainCache[s] = a
		return a, nil
	}

	for _, dc := range config.Domains {
		for _, rawRec := range dc.RawRecords {
			filePos := FixPosition(rawRec.FilePos)
			typeName := rawRec.Type
			//fmt.Printf("DEBUG: dc.Name=%s raw.Type=%s filepos=%s fp=%s\n", dc.Name, rawRec.Type, rawRec.FilePos, filePos)

			//fmt.Printf("DEBUG: IsBuilder(%q) = %v\n", typeName, IsBuilder(typeName))
			// NB(tlim): We check if something is a builder first because LOC could be a builder or a dnsrdatav2.
			if IsBuilder(typeName) {
				records, err := dc.runBuilder(typeName, rawRec.TTL, rawRec.Args, rawRec.SubDomain)
				if err != nil {
					return err
				}
				for _, record := range records {
					record.FilePos = filePos
				}
				// Generation complete!  Append it.
				dc.Records = append(dc.Records, records...)
			} else {
				typeNum, err := dnsutilv2.StringToType(typeName)
				if err != nil {
					return fmt.Errorf("unknown record type at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
				}

				// Apply D_EXTEND() subdomain label rewriting (in Go, on
				// post-IDNA strings). Excluded types keep their label as-is.
				// The subdomain is converted to IDNA (punycode) once and reused
				// for the label, the target origin, and rec.SubDomain.
				subdomain := rawRec.SubDomain
				if subdomainExcludedType(typeName) {
					subdomain = ""
				} else {
					subdomain, err = subdomainToASCII(subdomain)
					if err != nil {
						return fmt.Errorf("subdomain error at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
					}
				}
				label, err := dc.LabelFromDnsconfigjsSubdomain(rawRec.Args[0].(string), subdomain)
				if err != nil {
					return fmt.Errorf("label error at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
				}

				mm, err := mergeMetas(rawRec.Metas)
				if err != nil {
					return fmt.Errorf("metadata error at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
				}

				// ttl := rawRec.TTL
				// if ttl != 0 {
				// 	fmt.Printf("TTL = %d\n", ttl)
				// }
				rec, err := dc.NewRecordConfigFromDnsconfigjs(label, rawRec.TTL, typeNum, rawRec.Args[1:], mm, subdomain)
				if err != nil {
					return fmt.Errorf("ImportRawRecords error at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
				}
				rec.FilePos = filePos
				rec.SubDomain = subdomain
				if rec.Metadata, err = mergeMetas(rawRec.Metas); err != nil {
					return fmt.Errorf("metadata error at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
				}

				// The stutter check catches labels that mistakenly include the
				// zone name. It is skipped when the record opts out via
				// skip_fqdn_check, and for reverse zones, where labels are
				// legitimately written as full reverse names (e.g. via REV())
				// and are reduced to relative form later by normalization.
				if rec.Metadata["skip_fqdn_check"] != "true" && !isReverseZone(dc.Name) && doesStutter(rec.Name, dc.Name) {
					return fmt.Errorf("stutter error at %s %s(%s)", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args))
				}

				// Conversion complete!  Append it.
				dc.AddRecordConfig(rec)
			}

			// We're never going to see this rawRec again. Free its Args.
			clear(rawRec.Args)
			rawRec.Args = nil
		}

		// We're never goign to see these RawRecords again. Let them go.
		dc.RawRecords = nil
	}

	return nil
}

// FindDomain returns the *DomainConfig for domain query in config.
func (config *DNSConfig) FindDomain(query string) *DomainConfig {
	for _, b := range config.Domains {
		if b.Name == query {
			return b
		}
	}
	return nil
}

// RegistrarConfig describes a registrar.
type RegistrarConfig struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Metadata json.RawMessage `json:"meta,omitempty"`
}

// DNSProviderConfig describes a DNS service provider.
type DNSProviderConfig struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Metadata json.RawMessage `json:"meta,omitempty"`
}

// Nameserver describes a nameserver.
type Nameserver struct {
	Name string `json:"name"` // Normalized to a FQDN with NO trailing "."
	// NB(tlim): DomainConfig.Nameservers are stored WITH a trailing "." (Sorry!)
}

// FIXME(tal): In hindsight the Nameserver struct is overkill. We
// could have just used string.  Currently there are very few places
// that refer to the .Name field directly.  All new code should use
// ToNameservers/ToNameserversStripTD and NameserversToStrings to make
// future refactoring easier.  See
// https://github.com/DNSControl/dnscontrol/issues/577

func (n *Nameserver) String() string {
	return n.Name
}

// ToNameservers turns a list of strings into a list of Nameservers.
// It is an error if any string has a trailing dot. Either remove the
// trailing dot before you call this or (much preferred) use ToNameserversStripTD.
func ToNameservers(nss []string) ([]*Nameserver, error) {
	nservers := []*Nameserver{}
	for _, ns := range nss {
		if strings.HasSuffix(ns, ".") {
			return nil, errors.New("provider code leaves trailing dot on nameserver")
			// If you see this error, maybe the provider should call
			// ToNameserversStripTD instead.
		}
		nservers = append(nservers, &Nameserver{Name: ns})
	}
	return nservers, nil
}

// ToNameserversStripTD is like ToNameservers but strips the trailing
// dot from each item. It is an error if there is no trailing dot.
func ToNameserversStripTD(nss []string) ([]*Nameserver, error) {
	nservers := []*Nameserver{}
	for _, ns := range nss {
		if !strings.HasSuffix(ns, ".") {
			return nil, fmt.Errorf("provider code already removed nameserver trailing dot (%v)", ns)
			// If you see this error, maybe the provider should call ToNameservers instead.
		}
		nservers = append(nservers, &Nameserver{Name: ns[0 : len(ns)-1]})
	}
	return nservers, nil
}

// NameserversToStrings constructs a list of strings from *Nameserver structs.
func NameserversToStrings(nss []*Nameserver) (s []string) {
	for _, ns := range nss {
		s = append(s, ns.Name)
	}
	return s
}

// Correction is anything that can be run. Implementation is up to the specific provider.
type Correction struct {
	F   func() error `json:"-"`
	Msg string
}

// PostProcess performs and post-processing required after running dnsconfig.js and loading the result.
func (config *DNSConfig) PostProcess() error {
	for _, domain := range config.Domains {
		domain.PostProcess()
		// No need to call FixLegacyDC here. These records were created from dnsconfig.js, not from a provider.
	}
	return nil
}
