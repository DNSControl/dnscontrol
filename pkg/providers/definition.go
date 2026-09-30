package providers

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// InitializableProvider initializes a fresh provider instance. Implementations
// must populate the receiver, not replace it with a shared instance.
type InitializableProvider interface {
	Initialize(map[string]string, json.RawMessage, *CreateOptions) error
}

// RecordAuditingProvider validates records without credentials or initialization.
// AuditRecords must neither depend on nor mutate receiver state.
type RecordAuditingProvider interface {
	AuditRecords(models.Records) []error
}

// Initializer allocates and initializes a fresh instance, returning nil on error.
type Initializer func(map[string]string, json.RawMessage, *CreateOptions) (any, error)

// Definition describes a provider implementation, never a configured account.
// providers.Register is the main entry point. It registers the provider, generates
// the derived fields, and cross-checks for errors.
// Definitions and their nested metadata are read-only after registration.
type Definition struct {
	// FriendlyName is the brand name ("Google", not "GCLOUD")
	FriendlyName string

	// Aliases (optional) is an optional list of aliases. For example, if
	// we rename GCLOUD to GOOGLEDNS, we would make GCLOUD an alias to
	// support legacy configurations.
	Aliases []string

	// Maintainer is the github username of the maintainer.
	Maintainer string

	// DefaultTTL (optional) minimum TTL when `get-zone` writes .js files
	DefaultTTL uint32

	// CredFields is a description of the creds.json fields for this provider. Used by "init".
	CredFields []CredsField

	// DocsURL see CredsMetadata
	DocsURL string
	// PortalURL see CredsMetadata
	PortalURL string
	// Notes CredsMetadata
	Notes string
	// PostWrite CredsMetadata
	PostWrite func(map[string]string) error

	// Nil capability notes fall back to Features. Explicit notes take precedence.
	CanAutoDNSSEC          *DocumentationNote
	CanConcur              *DocumentationNote
	CanUseDSForChildren    *DocumentationNote
	DocDualHost            *DocumentationNote
	DocOfficiallySupported *DocumentationNote
	RecordIdentity         RecordIdentityFunc
	Features               DocumentationNotes

	// Calculated at providers.Register time. Providers must leave these fields unset.
	TypeName           string
	ImplementationType reflect.Type
	Kind               ProviderKind
	Initializer        Initializer
	RecordAuditor      RecordAuditor
	CanGetZones        bool
	DocCreateDomains   bool
	DerivedFeatures    DocumentationNotes
}

// Only migrated providers appear here. Legacy registrations continue to use the
// compatibility registries until all providers and consumers have migrated.
// Canonical names and aliases point directly to the same definition.
var definitions = map[string]*Definition{}

// Register registers a concrete pointer-to-struct implementation. Invalid or
// conflicting declarations panic during startup, before any provider is used.
// Call it during package initialization, before concurrent registry reads.
// Registration inspects interfaces; it never calls Initialize.
// Supplied slices, maps, and notes must not be modified after registration.
func Register[T InitializableProvider](name string, definition Definition) {
	typ := reflect.TypeFor[T]()
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("provider %q: implementation must be a pointer to a concrete struct, got %v", name, typ))
	}
	// Validate FriendlyName
	if strings.TrimSpace(definition.FriendlyName) == "" {
		panic(fmt.Sprintf("provider %q: FriendlyName is required", name))
	}
	// Validate Features
	for capability, note := range definition.Features {
		if note == nil {
			panic(fmt.Sprintf("provider %q: Features[%v] must not be nil", name, capability))
		}
	}
	// Make sure caller didn't set fields that are generated.
	if definition.TypeName != "" || definition.ImplementationType != nil || definition.Kind != 0 ||
		definition.Initializer != nil || definition.RecordAuditor != nil || definition.CanGetZones ||
		definition.DocCreateDomains || definition.DerivedFeatures != nil {
		panic(fmt.Sprintf("provider %q: derived definition fields must be left unset", name))
	}

	// Verify no duplicate/invalid names.
	names := append([]string{name}, definition.Aliases...)
	seen := map[string]bool{}
	for _, n := range names {
		if strings.TrimSpace(n) == "" || n == "-" || seen[n] || providerNameInUse(n) {
			panic(fmt.Sprintf("provider %q: invalid or already registered name %q", name, n))
		}
		seen[n] = true
	}

	def := &definition
	def.TypeName = name
	def.ImplementationType = typ
	if typ.Implements(reflect.TypeFor[DNSServiceProvider]()) {
		def.Kind |= KindDNS
	}
	if typ.Implements(reflect.TypeFor[Registrar]()) {
		def.Kind |= KindRegistrar
	}
	if def.Kind == 0 {
		panic(fmt.Sprintf("provider %q: implementation supports neither DNS nor registrar operations", name))
	}
	if typ.Implements(reflect.TypeFor[RecordAuditingProvider]()) {
		// This receiver is separate from every runtime account instance.
		auditor := reflect.New(typ.Elem()).Interface().(RecordAuditingProvider)
		def.RecordAuditor = auditor.AuditRecords
	} else if def.Kind.Has(KindDNS) {
		panic(fmt.Sprintf("provider %q: DNS implementations must implement RecordAuditingProvider", name))
	}
	def.CanGetZones = typ.Implements(reflect.TypeFor[ZoneLister]())
	def.DocCreateDomains = typ.Implements(reflect.TypeFor[ZoneCreator]())
	def.Initializer = func(config map[string]string, meta json.RawMessage, options *CreateOptions) (any, error) {
		resolved := options.WithDefaults()
		if role := resolved.RequestedRole; role != 0 &&
			(role != KindDNS && role != KindRegistrar || !def.Kind.Has(role)) {
			return nil, fmt.Errorf("provider %q does not support requested role %d", name, role)
		}
		instance := reflect.New(typ.Elem()).Interface().(T)
		if err := instance.Initialize(config, meta, &resolved); err != nil {
			return nil, err
		}
		return instance, nil
	}
	def.DerivedFeatures = maps.Clone(def.Features)
	if def.DerivedFeatures == nil {
		def.DerivedFeatures = DocumentationNotes{}
	}
	for capability, note := range map[Capability]*DocumentationNote{
		CanAutoDNSSEC: def.CanAutoDNSSEC, CanConcur: def.CanConcur,
		CanUseDSForChildren: def.CanUseDSForChildren, DocDualHost: def.DocDualHost,
		DocOfficiallySupported: def.DocOfficiallySupported,
	} {
		if note != nil {
			def.DerivedFeatures[capability] = note
		}
	}
	for capability, supported := range map[Capability]bool{
		CanGetZones: def.CanGetZones, DocCreateDomains: def.DocCreateDomains,
	} {
		// Preserve annotations without overwriting the supplied feature note.
		var note DocumentationNote
		if legacy := def.DerivedFeatures[capability]; legacy != nil {
			note = *legacy
		}
		note.HasFeature = supported
		note.Unimplemented = false
		def.DerivedFeatures[capability] = &note
	}

	for _, n := range names {
		definitions[n] = def
	}
	publishLegacyDefinition(def)
}

func providerNameInUse(name string) bool {
	_, definition := definitions[name]
	_, dns := DNSProviderTypes[name]
	_, registrar := RegistrarTypes[name]
	_, creds := CredsMetadataByType[name]
	_, maintainer := ProviderMaintainers[name]
	_, ttl := ProviderDefaultTTLs[name]
	return definition || dns || registrar || creds || maintainer || ttl
}

// rejectUnifiedRegistration prevents mixing legacy writes with a completed
// definition, including aliases whose canonical provider has a different role.
func rejectUnifiedRegistration(name string) {
	if _, ok := definitions[name]; ok {
		panic(fmt.Sprintf("provider %q is already registered with Register", name))
	}
}

func canonicalProviderName(name string) string {
	if def, ok := definitions[name]; ok {
		return def.TypeName
	}
	return name
}

// GetDefinition returns the stored definition for a migrated provider. Canonical
// names and aliases return the same pointer. The definition and its nested
// metadata must be treated as read-only.
func GetDefinition(name string) (*Definition, bool) {
	def, ok := definitions[name]
	return def, ok
}

// AllDefinitions returns migrated canonical definitions sorted by TypeName.
// The returned pointers and their nested metadata must be treated as read-only.
// Legacy providers remain available through the compatibility registries.
func AllDefinitions() []*Definition {
	names := slices.Sorted(maps.Keys(definitions))
	result := make([]*Definition, 0, len(names))
	for _, name := range names {
		if def := definitions[name]; name == def.TypeName {
			result = append(result, def)
		}
	}
	return result
}

func definitionCredsMetadata(def *Definition) CredsMetadata {
	return CredsMetadata{
		TypeName: def.TypeName, DisplayName: def.FriendlyName, Kind: def.Kind,
		DocsURL: def.DocsURL, PortalURL: def.PortalURL, Notes: def.Notes,
		Fields: def.CredFields, PostWrite: def.PostWrite,
	}
}

// publishLegacyDefinition maintains canonical-only compatibility views for
// consumers not yet migrated. These exported maps must be treated as read-only;
// accessor/factory calls use the private definition as their source of truth.
func publishLegacyDefinition(def *Definition) {
	name := def.TypeName
	if def.Kind.Has(KindDNS) {
		DNSProviderTypes[name] = DspFuncs{
			Initializer: func(config map[string]string, meta json.RawMessage) (DNSServiceProvider, error) {
				return CreateDNSProvider(name, config, meta)
			},
			InitializerWithOptions: func(config map[string]string, meta json.RawMessage, options CreateOptions) (DNSServiceProvider, error) {
				return CreateDNSProvider(name, config, meta, func(o *CreateOptions) { *o = options })
			},
			RecordAuditor: def.RecordAuditor, RecordIdentity: def.RecordIdentity,
		}
	}
	if def.Kind.Has(KindRegistrar) {
		RegistrarTypes[name] = func(config map[string]string) (Registrar, error) {
			return CreateRegistrar(name, config)
		}
	}
	CredsMetadataByType[name] = definitionCredsMetadata(def)
	if def.Maintainer != "" {
		ProviderMaintainers[name] = def.Maintainer
	}
	if def.DefaultTTL != 0 {
		ProviderDefaultTTLs[name] = def.DefaultTTL
	}
	unwrapProviderCapabilities(name, []ProviderMetadata{def.DerivedFeatures})
}
