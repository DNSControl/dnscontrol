package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// Registrar is an interface for a domain registrar. It can return a list of needed corrections to be applied in the future. Implement this only if the provider is a "registrar" (i.e. can update the NS records of the parent to a domain).
type Registrar interface {
	models.Registrar
}

// DNSServiceProvider is able to generate a set of corrections that need to be made to correct records for a domain. Implement this only if the provider is a DNS Service Provider (can update records in a DNS zone).
type DNSServiceProvider interface {
	models.DNSProvider
}

// ZoneCreator should be implemented by providers that have the ability to create zones
// (used for automatically creating zones if they don't exist).
type ZoneCreator interface {
	EnsureZoneExists(dc *models.DomainConfig) error
}

// ZoneLister should be implemented by providers that have the
// ability to list the zones they manage. This facilitates using the
// "get-zones" command for "all" zones.
type ZoneLister interface {
	ListZones() ([]string, error)
}

// RegistrarInitializer is a function to create a registrar. Function will be passed the unprocessed json payload from the configuration file for the given provider.
type RegistrarInitializer func(map[string]string) (Registrar, error)

// RegistrarTypes stores initializer for each registrar.
var RegistrarTypes = map[string]RegistrarInitializer{}

// DspInitializer is a function to create a DNS service provider. Function will be passed the unprocessed json payload from the configuration file for the given provider.
type DspInitializer func(map[string]string, json.RawMessage) (DNSServiceProvider, error)

// RecordAuditor is a function that verifies that all the records
// are supportable by this provider. It returns a list of errors
// detailing records that this provider can not support.
type RecordAuditor func([]*models.RecordConfig) []error

// DspFuncs lists functions registered with a provider.
type DspFuncs struct {
	Initializer   DspInitializer
	RecordAuditor RecordAuditor
}

// DNSProviderTypes stores initializer for each DSP.
var DNSProviderTypes = map[string]DspFuncs{}

// RegisterRegistrarType adds a registrar type to the registry by providing a suitable initialization function.
func RegisterRegistrarType(name string, init RegistrarInitializer, pm ...ProviderMetadata) {
	if _, ok := RegistrarTypes[name]; ok {
		log.Fatalf("Cannot register registrar type %q multiple times", name)
	}
	RegistrarTypes[name] = init
	unwrapProviderCapabilities(name, pm)
}

// RegisterDomainServiceProviderType adds a dsp to the registry with the given initialization function.
func RegisterDomainServiceProviderType(name string, fns DspFuncs, pm ...ProviderMetadata) {
	if _, ok := DNSProviderTypes[name]; ok {
		log.Fatalf("Cannot register registrar type %q multiple times", name)
	}
	DNSProviderTypes[name] = fns

	unwrapProviderCapabilities(name, pm)
}

// ProviderMaintainers stores the GitHub usernames of maintainers for each provider.
var ProviderMaintainers = map[string]string{}

// RegisterMaintainer registers the GitHub username of the maintainer for a provider.
func RegisterMaintainer(
	providerName string,
	gitHubUsername string,
) {
	ProviderMaintainers[providerName] = gitHubUsername
}

// ProviderDefaultTTLs stores the default TTL for each provider.
var ProviderDefaultTTLs = map[string]uint32{}

// RegisterDefaultTTL registers a default TTL for a provider.
// This is used by get-zones to determine the DefaultTTL when generating output.
func RegisterDefaultTTL(providerName string, defaultTTL uint32) {
	ProviderDefaultTTLs[providerName] = defaultTTL
}

// GetDefaultTTL returns the default TTL for a provider, or 0 if not registered.
func GetDefaultTTL(providerName string) uint32 {
	return ProviderDefaultTTLs[providerName]
}

// CreateRegistrar initializes a registrar instance from given credentials.
func CreateRegistrar(rType string, config map[string]string) (Registrar, error) {
	var err error
	rType, err = beCompatible(rType, config)
	if err != nil {
		return nil, err
	}

	initer, ok := RegistrarTypes[rType]
	if !ok {
		return nil, fmt.Errorf("no such registrar type: %q", rType)
	}
	return initer(config)
}

// CreateDNSProvider initializes a dns provider instance from given credentials.
func CreateDNSProvider(providerTypeName string, config map[string]string, meta json.RawMessage) (DNSServiceProvider, error) {
	var err error
	providerTypeName, err = beCompatible(providerTypeName, config)
	if err != nil {
		return nil, err
	}

	p, ok := DNSProviderTypes[providerTypeName]
	if !ok {
		return nil, fmt.Errorf("no such DNS service provider: %q", providerTypeName)
	}
	return p.Initializer(config, meta)
}

// beCompatible looks up.
func beCompatible(n string, config map[string]string) (string, error) {
	// Pre 4.0: If n is a placeholder, substitute the TYPE from creds.json.
	// 4.0: Require TYPE from creds.json.

	ct := config["TYPE"]
	// If a placeholder value was specified...
	if n == "" || n == "-" {
		// But no TYPE exists in creds.json...
		if ct == "" {
			return "-", errors.New("creds.json entry missing TYPE field")
		}
		// Otherwise, use the value from creds.json.
		return ct, nil
	}

	// Pre 4.0: The user specified the name manually.
	// Cross check to detect user-error.
	if ct != "" && n != ct {
		return "", fmt.Errorf("creds.json entry mismatch: specified=%q TYPE=%q", n, ct)
	}
	// Seems like the user did it the right way. Return the original value.
	return n, nil

	// NB(tlim): My hope is that in 4.0 this entire function will simply be the
	// following, but I may be wrong:
	// return config["TYPE"], nil
}

// AuditRecords calls the RecordAudit function for a provider.
func AuditRecords(dType string, rcs models.Records) []error {
	p, ok := DNSProviderTypes[dType]
	if !ok {
		return []error{fmt.Errorf("unknown DNS service provider type: %q", dType)}
	}
	if p.RecordAuditor == nil {
		return []error{fmt.Errorf("DNS service provider type %q has no RecordAuditor", dType)}
	}
	return p.RecordAuditor(rcs)
}

// None is a basic provider type that does absolutely nothing. Can be useful as a placeholder for third parties or unimplemented providers.
type None struct{}

// GetRegistrarCorrections returns corrections to update registrars.
func (n None) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	return nil, nil
}

// GetNameservers returns the current nameservers for a domain.
func (n None) GetNameservers(string) ([]*models.Nameserver, error) {
	return nil, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (n None) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	return nil, nil
}

// GetZoneRecordsCorrections gets the records of a zone and returns them in RecordConfig format.
func (n None) GetZoneRecordsCorrections(dc *models.DomainConfig, records models.Records) ([]*models.Correction, int, error) {
	return nil, 0, nil
}

var featuresNone = DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	CanConcur: Can(),
}

func init() {
	RegisterRegistrarType("NONE", func(map[string]string) (Registrar, error) {
		return None{}, nil
	}, featuresNone)
	RegisterCredsMetadata("NONE", CredsMetadata{
		DisplayName: "No registrar",
		Kind:        KindRegistrar,
		Notes:       "Use NONE when you do not want DNSControl to manage nameserver delegation (that is, the nameservers for this domain will be managed manually).",
	})
}

// CustomRType stores an rtype that is only valid for this DSP.
type CustomRType struct {
	Name     string
	Provider string
	RealType string
}

// RegisterCustomRecordType registers a record type that is only valid for one provider.
// provider is the registered type of provider this is valid with
// name is the record type as it will appear in the js. (should be something like $PROVIDER_FOO)
// realType is the record type it will be replaced with after validation.
func RegisterCustomRecordType(name, provider, realType string) {
	customRecordTypes[name] = &CustomRType{Name: name, Provider: provider, RealType: realType}
}

// GetCustomRecordType returns a registered custom record type, or nil if none.
func GetCustomRecordType(rType string) *CustomRType {
	return customRecordTypes[rType]
}

var customRecordTypes = map[string]*CustomRType{}
