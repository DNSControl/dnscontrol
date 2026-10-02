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

// Initializer allocates and initializes a fresh instance, returning nil on error.
type Initializer func(map[string]string, json.RawMessage, *CreateOptions) (any, error)

// Definition describes a provider implementation, never a configured account.
// providers.Register is the main entry point. It registers the provider, generates
// the derived fields, and cross-checks for errors.
// Definitions and their nested metadata are read-only to callers. Framework
// finalization completes derived type capabilities before concurrent reads.
type Definition struct {
	// FriendlyName is the brand name ("Google", not "GCLOUD")
	FriendlyName string

	// Aliases (optional) is an optional list of aliases. For example, if
	// we rename CLOUDFLAREAPI to CLOUDFLARE, we would make CLOUDFLAREAPI an alias to
	// support legacy configurations.
	Aliases []string

	// Maintainer is the github username of the maintainer.
	Maintainer string

	// DefaultTTL (optional) minimum TTL when `get-zone` writes .js files
	DefaultTTL uint32

	// CredFields is a description of the creds.json fields for this provider. Used by "init".
	CredFields []CredsField

	// DocsURL optionally overrides https://docs.dnscontrol.org/provider/<lowercase name>
	// for legacy documentation paths. An override matching the default is an error.
	// Register fills an empty value with the default URL.
	DocsURL string
	// VendorAPIDocURL optionally links to the vendor's API documentation.
	VendorAPIDocURL string
	// PortalURL points to where a user can obtain API credentials.
	PortalURL string
	// Notes is optional onboarding text shown before prompting for credentials.
	Notes string
	// PostWrite prepares local resources after the wizard writes creds.json.
	PostWrite func(map[string]string) error

	// Nil capability notes fall back to Features. Explicit notes take precedence.
	CanAutoDNSSEC          *DocumentationNote
	CanConcur              *DocumentationNote
	CanUseDSForChildren    *DocumentationNote
	DocDualHost            *DocumentationNote
	DocOfficiallySupported *DocumentationNote
	RecordIdentity         RecordIdentityFunc
	Features               DocumentationNotes

	// SupportedTypes is an exhaustive declaration of record support. Nil means
	// Default (A, AAAA, CAA, CNAME, MX, NS, SRV, TXT), except that non-nil
	// Features retains legacy validation. A non-nil empty slice declares no
	// support beyond Features.
	// Internal commands such as IMPORT_TRANSFORM do not belong in this list.
	SupportedTypes []string

	typeSelectors   []typeSelector
	exhaustiveTypes bool

	// Calculated by Register and Finalize. Providers must leave these fields unset.
	TypeName           string
	ImplementationType reflect.Type
	Kind               ProviderKind
	Initializer        Initializer
	RecordAuditor      RecordAuditor
	CanGetZones        bool
	DocCreateDomains   bool
	DerivedFeatures    DocumentationNotes
}

// Provider implementations register here.
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
		if strings.TrimSpace(n) == "" || n == "-" || seen[n] || definitions[n] != nil {
			panic(fmt.Sprintf("provider %q: invalid or already registered name %q", name, n))
		}
		seen[n] = true
	}

	def := &definition
	def.TypeName = name
	def.exhaustiveTypes = def.SupportedTypes != nil || def.Features == nil
	if err := def.compileTypeSelectors(); err != nil {
		panic(err)
	}
	defaultDocsURL := "https://docs.dnscontrol.org/provider/" + strings.ToLower(name)
	if def.DocsURL == defaultDocsURL {
		panic(fmt.Sprintf("provider %q: DocsURL override matches derived URL %q; omit it", name, defaultDocsURL))
	}
	if def.DocsURL == "" {
		def.DocsURL = defaultDocsURL
	}
	def.ImplementationType = typ
	if typ.Implements(reflect.TypeFor[DNSServiceProvider]()) {
		def.Kind |= KindDNS
		// This uninitialized receiver is separate from every runtime account.
		auditor := reflect.New(typ.Elem()).Interface().(models.DNSProvider)
		def.RecordAuditor = auditor.AuditRecords
	}
	if typ.Implements(reflect.TypeFor[Registrar]()) {
		def.Kind |= KindRegistrar
	}
	if def.Kind == 0 {
		panic(fmt.Sprintf("provider %q: implementation supports neither DNS nor registrar operations", name))
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
	definitionGeneration++
}

// CanonicalName resolves a provider-type alias. Unknown names are returned
// unchanged so callers can retain their existing validation and diagnostics.
func CanonicalName(name string) string {
	if def, ok := definitions[name]; ok {
		return def.TypeName
	}
	return name
}

// GetDefinition returns the stored provider definition. Canonical
// names and aliases return the same pointer. The definition and its nested
// metadata must be treated as read-only.
func GetDefinition(name string) (*Definition, bool) {
	mustFinalize()
	def, ok := definitions[name]
	return def, ok
}

// AllDefinitions returns canonical definitions sorted by TypeName.
// The returned pointers and their nested metadata must be treated as read-only.
func AllDefinitions() []*Definition {
	mustFinalize()
	return allDefinitions()
}

func allDefinitions() []*Definition {
	names := slices.Sorted(maps.Keys(definitions))
	result := make([]*Definition, 0, len(names))
	for _, name := range names {
		// Only copy the canonical item. Not aliases.
		if def := definitions[name]; name == def.TypeName {
			result = append(result, def)
		}
	}
	return result
}
