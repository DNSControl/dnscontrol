package providers

import (
	"log"
)

// ProviderKind is a bitmask describing the capabilities a provider offers
// from the perspective of the DNSControl configuration.
type ProviderKind uint8

const (
	// KindDNS marks a provider that can host DNS records.
	KindDNS ProviderKind = 1 << iota
	// KindRegistrar marks a provider that can act as a domain registrar.
	KindRegistrar
)

// Has reports whether the given kind bit is set.
func (kind ProviderKind) Has(other ProviderKind) bool {
	return kind&other == other
}

// CredsField describes a single key/value pair that belongs in the
// provider entry inside creds.json.
type CredsField struct {
	// Key is the exact name of the field as it should appear inside the
	// creds.json provider entry (for example "apitoken" or "KeyId").
	Key string
	// Label is the human readable text shown when prompting.
	Label string
	// Help is a short sentence explaining the field.
	Help string
	// Secret masks the input when prompting. Ignored when Multiline is
	// also set, since the external editor does not mask input.
	Secret bool
	// Multiline opens the user's $EDITOR so values like PEM blocks with
	// embedded newlines can be entered in full.
	Multiline bool
	// Required makes the init command reject an empty value.
	Required bool
	// Default suggests a value. Presented as the pre filled answer.
	Default string
	// EnvVar, when set, is used as the default if the environment variable is
	// present. It overrides Default.
	EnvVar string
	// Choices restricts input to one of the listed values.
	Choices []string
	// Validator can reject an entered value with an explanatory error.
	Validator func(string) error
	// Internal marks the field as a UI selector whose answer drives
	// ShowIf logic on later fields. Internal answers are not written to
	// creds.json.
	//
	// Convention: prefix the Key with an underscore (for example
	// "_authMethod") so it is visually distinct from real creds.json
	// keys when reading metadata registrations.
	Internal bool
	// ShowIf restricts when this field is shown. Each entry maps an
	// earlier field's Key to the value that must be selected for this
	// field to appear. An empty map means always show.
	ShowIf map[string]string
}

// CredsMetadata documents the creds.json layout for a single provider type
// plus the onboarding hints a human needs to obtain the values.
type CredsMetadata struct {
	// TypeName matches the name passed to RegisterDomainServiceProviderType
	// or RegisterRegistrarType (for example "CLOUDFLAREAPI").
	TypeName string
	// DisplayName is a friendly name for the provider (for example
	// "Cloudflare").
	DisplayName string
	// Kind indicates which registry maps the provider appears in.
	Kind ProviderKind
	// DocsURL points to the provider documentation page.
	DocsURL string
	// PortalURL is the URL where a human can create the API credential.
	PortalURL string
	// Fields lists the creds.json keys in the order the init command should
	// prompt for them.
	Fields []CredsField
	// Notes is an optional block of text shown once before prompting.
	Notes string
	// PostWrite, if set, is called after the wizard has written
	// creds.json so the provider can prepare any local resources it
	// needs. BIND uses this to create the zone files directory.
	PostWrite func(fields map[string]string) error
}

// CredsMetadataByType stores every registered CredsMetadata keyed by
// TypeName.
var CredsMetadataByType = map[string]CredsMetadata{}

// RegisterCredsMetadata records the creds.json metadata for a provider.
// It is safe to call from a provider init() alongside RegisterMaintainer.
func RegisterCredsMetadata(name string, meta CredsMetadata) {
	if _, ok := CredsMetadataByType[name]; ok {
		log.Fatalf("Cannot register creds metadata for %q multiple times", name)
	}
	if meta.TypeName == "" {
		meta.TypeName = name
	}
	CredsMetadataByType[name] = meta
}

// GetCredsMetadata returns the metadata registered for the given provider
// type, or a zero value and false when nothing is registered.
func GetCredsMetadata(name string) (CredsMetadata, bool) {
	meta, found := CredsMetadataByType[name]
	return meta, found
}
