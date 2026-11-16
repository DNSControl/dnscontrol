package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
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

// validTagRe matches the tags permitted in "example.com!tag". The empty tag is permitted.
var validTagRe = regexp.MustCompile(`^([A-Za-z0-9_-][A-Za-z0-9_.-]*)?$`)

// PostProcess performs and post-processing required after running dnsconfig.js and loading the result.
func (config *DNSConfig) PostProcess() error {
	for _, domain := range config.Domains {
		// A domain name with a trailing dot is invalid (NewDomainConfig
		// rejects the same) and panics downstream in SetLabel.  Reject it
		// here so that user input can never crash dnscontrol.
		if strings.HasSuffix(domain.Name, ".") {
			return fmt.Errorf("domain name %q must not end with a dot", domain.Name)
		}
		// A domain name with an empty label (a leading dot or consecutive
		// dots, as in "../example.com") is not a valid DNS name.  Providers
		// use the name to build filenames and URLs, so reject it here.
		if slices.Contains(strings.Split(domain.Name, "."), "") {
			return fmt.Errorf("domain name %q must not contain an empty label", domain.Name)
		}
		// The tag (the "tag" in "example.com!tag") is used by providers to
		// build filenames, so restrict it to a conservative character set.
		if !validTagRe.MatchString(domain.Tag) {
			return fmt.Errorf("domain %q: tag %q may only contain letters, digits, hyphens, underscores, and dots, and must not start with a dot", domain.Name, domain.Tag)
		}
		// dnsconfig.js creates records only via the record constructors
		// (A(), TXT(), ...), which arrive as .RawRecords and are imported
		// by ImportRawRecords().  RecordConfig.rdata is unexported, so a
		// record injected via a JSON "records" or "recordsabsent" field
		// can never be well-formed.  Reject malformed records here so
		// they cannot crash dnscontrol later.
		for i, rec := range domain.Records {
			if rec == nil || rec.rdata == nil {
				return fmt.Errorf("domain %s: .Records[%d] is nil or was not created by a record constructor", domain.Name, i)
			}
		}
		for i, rec := range domain.EnsureAbsent {
			if rec == nil || rec.rdata == nil {
				return fmt.Errorf("domain %s: .EnsureAbsent[%d] is nil or was not created by a record constructor", domain.Name, i)
			}
		}
		domain.PostProcess()
	}
	return nil
}
