package providers

import (
	"encoding/json"
	"errors"
	"fmt"

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

// GetRecordIdentity returns the provider's RecordIdentity function, or nil if
// the provider does not declare one.
func GetRecordIdentity(dType string) RecordIdentityFunc {
	if def, ok := definitions[dType]; ok {
		return def.RecordIdentity
	}
	return nil
}

// GetDefaultTTL returns the provider's default TTL, or 0 if not registered.
func GetDefaultTTL(providerName string) uint32 {
	if def, ok := GetDefinition(providerName); ok {
		return def.DefaultTTL
	}
	return 0
}

// CreateRegistrar initializes a registrar instance from given credentials.
func CreateRegistrar(rType string, config map[string]string, opts ...CreateOption) (Registrar, error) {
	var err error
	rType, err = beCompatible(rType, config)
	if err != nil {
		return nil, err
	}
	def, ok := GetDefinition(rType)
	if !ok || !def.Kind.Has(KindRegistrar) {
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

// CreateDNSProvider initializes a dns provider instance from given credentials.
func CreateDNSProvider(providerTypeName string, config map[string]string, meta json.RawMessage, opts ...CreateOption) (DNSServiceProvider, error) {
	var err error
	providerTypeName, err = beCompatible(providerTypeName, config)
	if err != nil {
		return nil, err
	}
	options := newCreateOptions(opts)
	options.RequestedRole = KindDNS
	def, ok := GetDefinition(providerTypeName)
	if !ok || !def.Kind.Has(KindDNS) {
		return nil, fmt.Errorf("no such DNS service provider: %q", providerTypeName)
	}
	instance, err := def.Initializer(config, meta, &options)
	if err != nil {
		return nil, err
	}
	return instance.(DNSServiceProvider), nil
}

// beCompatible resolves explicit or credential-supplied types, accepting aliases
// while preserving the legacy explicit-type fallback when TYPE is absent.
func beCompatible(n string, config map[string]string) (string, error) {
	ct := config["TYPE"]
	// If a placeholder value was specified...
	if n == "" || n == "-" {
		// But no TYPE exists in creds.json...
		if ct == "" {
			return "-", errors.New("creds.json entry missing TYPE field")
		}
		// Otherwise, use the value from creds.json.
		return CanonicalName(ct), nil
	}

	// Cross-check an explicit type against credentials when both are present.
	if ct != "" && CanonicalName(n) != CanonicalName(ct) {
		return "", fmt.Errorf("creds.json entry mismatch: specified=%q TYPE=%q", n, ct)
	}
	return CanonicalName(n), nil
}

// AuditRecords calls the RecordAudit function for a provider.
func AuditRecords(dType string, rcs models.Records) []error {
	if def, ok := definitions[dType]; ok {
		if !def.Kind.Has(KindDNS) {
			return []error{fmt.Errorf("provider %q does not support the DNS role", dType)}
		}
		return def.RecordAuditor(rcs)
	}
	return []error{fmt.Errorf("unknown DNS service provider type: %q", dType)}
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
