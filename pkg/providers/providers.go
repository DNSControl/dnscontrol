package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/DNSControl/dnscontrol/v5/models"
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

// DspInitializerWithOptions creates a DNS service provider with optional
// constructor dependencies such as a conversion observer.
type DspInitializerWithOptions func(map[string]string, json.RawMessage, CreateOptions) (DNSServiceProvider, error)

// RecordAuditor is a function that verifies that all the records
// are supportable by this provider. It returns a list of errors
// detailing records that this provider can not support.
type RecordAuditor func(models.Records) []error

// RecordIdentityFunc returns the identity text of a record: text that
// distinguishes two records which share a label, rType and RDATA but are stored
// by the provider as separate objects (DNSPod record lines, Route 53 routing
// policies). Validation uses it for duplicate detection, and validation runs
// before the provider has read the zone, so the function must depend only on
// the record it is given.
type RecordIdentityFunc func(*models.RecordConfig) string

// DspFuncs lists functions registered with a provider.
type DspFuncs struct {
	Initializer            DspInitializer
	InitializerWithOptions DspInitializerWithOptions
	RecordAuditor          RecordAuditor
	RecordIdentity         RecordIdentityFunc
}

// GetRecordIdentity returns the provider's RecordIdentity function, or nil if
// the provider does not declare one.
func GetRecordIdentity(dType string) RecordIdentityFunc {
	if def, ok := definitions[dType]; ok {
		return def.RecordIdentity
	}
	p, ok := DNSProviderTypes[dType]
	if !ok {
		return nil
	}
	return p.RecordIdentity
}

// DNSProviderTypes stores initializer for each DSP.
var DNSProviderTypes = map[string]DspFuncs{}

// RegisterRegistrarType adds a registrar type to the registry by providing a suitable initialization function.
func RegisterRegistrarType(name string, init RegistrarInitializer, pm ...ProviderMetadata) {
	rejectUnifiedRegistration(name)
	if _, ok := RegistrarTypes[name]; ok {
		log.Fatalf("Cannot register registrar type %q multiple times", name)
	}
	RegistrarTypes[name] = init
	unwrapProviderCapabilities(name, pm)
}

// RegisterDomainServiceProviderType adds a dsp to the registry with the given initialization function.
func RegisterDomainServiceProviderType(name string, fns DspFuncs, pm ...ProviderMetadata) {
	rejectUnifiedRegistration(name)
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
	rejectUnifiedRegistration(providerName)
	ProviderMaintainers[providerName] = gitHubUsername
}

// ProviderDefaultTTLs stores the default TTL for each provider.
var ProviderDefaultTTLs = map[string]uint32{}

// RegisterDefaultTTL registers a default TTL for a provider.
// This is used by get-zones to determine the DefaultTTL when generating output.
func RegisterDefaultTTL(providerName string, defaultTTL uint32) {
	rejectUnifiedRegistration(providerName)
	ProviderDefaultTTLs[providerName] = defaultTTL
}

// GetDefaultTTL returns the default TTL for a provider, or 0 if not registered.
func GetDefaultTTL(providerName string) uint32 {
	if def, ok := definitions[providerName]; ok {
		return def.DefaultTTL
	}
	return ProviderDefaultTTLs[providerName]
}

// CreateRegistrar initializes a registrar instance from given credentials.
func CreateRegistrar(rType string, config map[string]string, opts ...CreateOption) (Registrar, error) {
	var err error
	rType, err = beCompatible(rType, config)
	if err != nil {
		return nil, err
	}
	if def, ok := definitions[rType]; ok {
		if !def.Kind.Has(KindRegistrar) {
			return nil, fmt.Errorf("no such registrar type: %q", rType)
		}
		options := newCreateOptions(opts)
		options.RequestedRole = KindRegistrar
		instance, err := def.Initializer(config, nil, &options)
		if err != nil {
			return nil, err
		}
		return instance.(Registrar), nil
	}

	initer, ok := RegistrarTypes[rType]
	if !ok {
		return nil, fmt.Errorf("no such registrar type: %q", rType)
	}
	return initer(config)
}

// CreateDNSProvider initializes a dns provider instance from given credentials.
func CreateDNSProvider(providerTypeName string, config map[string]string, meta json.RawMessage, opts ...CreateOption) (DNSServiceProvider, error) {
	var err error
	providerTypeName, err = beCompatible(providerTypeName, config)
	if err != nil {
		return nil, err
	}
	options := newCreateOptions(opts)
	options.RequestedRole = KindDNS
	if def, ok := definitions[providerTypeName]; ok {
		if !def.Kind.Has(KindDNS) {
			return nil, fmt.Errorf("no such DNS service provider: %q", providerTypeName)
		}
		instance, err := def.Initializer(config, meta, &options)
		if err != nil {
			return nil, err
		}
		return instance.(DNSServiceProvider), nil
	}

	p, ok := DNSProviderTypes[providerTypeName]
	if !ok {
		return nil, fmt.Errorf("no such DNS service provider: %q", providerTypeName)
	}
	if p.InitializerWithOptions != nil {
		return p.InitializerWithOptions(config, meta, options)
	}
	provider, err := p.Initializer(config, meta)
	if err != nil {
		return nil, err
	}
	if setter, ok := provider.(ConversionObserverSetter); ok {
		setter.SetConversionObserver(options.ConversionObserver)
	}
	return provider, nil
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
		return canonicalProviderName(ct), nil
	}

	// Pre 4.0: The user specified the name manually.
	// Cross check to detect user-error.
	if ct != "" && canonicalProviderName(n) != canonicalProviderName(ct) {
		return "", fmt.Errorf("creds.json entry mismatch: specified=%q TYPE=%q", n, ct)
	}
	// Seems like the user did it the right way. Return the original value.
	return canonicalProviderName(n), nil

	// NB(tlim): My hope is that in 4.0 this entire function will simply be the
	// following, but I may be wrong:
	// return config["TYPE"], nil
}

// AuditRecords calls the RecordAudit function for a provider.
func AuditRecords(dType string, rcs models.Records) []error {
	if def, ok := definitions[dType]; ok {
		if !def.Kind.Has(KindDNS) {
			return []error{fmt.Errorf("provider %q does not support the DNS role", dType)}
		}
		return def.RecordAuditor(rcs)
	}
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

// Initialize needs no credentials or setup for the placeholder registrar.
func (*None) Initialize(map[string]string, json.RawMessage, *CreateOptions) error {
	return nil
}

func init() {
	Register[*None]("NONE", Definition{
		FriendlyName: "No registrar",
		CanConcur:    Can(),
		Notes:        "Use NONE when you do not want DNSControl to manage nameserver delegation (that is, the nameservers for this domain will be managed manually).",
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
