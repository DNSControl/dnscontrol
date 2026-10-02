package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// Tests use an isolated registry rather than editing entries published by Register.
func isolateDefinitions(t *testing.T) {
	t.Helper()
	oldDefs := definitions
	t.Cleanup(func() { definitions = oldDefs })
	definitions = map[string]*Definition{}
}

type definitionDNS struct {
	observerProvider
	account string
	meta    json.RawMessage
	options CreateOptions
}

var definitionInitializations int

func (p *definitionDNS) Initialize(config map[string]string, meta json.RawMessage, options *CreateOptions) error {
	definitionInitializations++
	p.account = config["account"]
	p.meta = slices.Clone(meta)
	p.options = options.WithDefaults()
	p.options.ConversionObserver.BeginToRC("initialize", nil)
	if config["fail"] != "" {
		return errors.New("initialization failed")
	}
	return nil
}

func (p *definitionDNS) AuditRecords(records models.Records) []error {
	if p.account != "" || p.options.ConversionObserver != nil {
		return []error{errors.New("auditor was initialized")}
	}
	if len(records) > 0 {
		return []error{errors.New("record rejected")}
	}
	return nil
}

type definitionDual struct{ definitionDNS }

func (*definitionDual) GetRegistrarCorrections(*models.DomainConfig) ([]*models.Correction, error) {
	return nil, nil
}
func (*definitionDual) ListZones() ([]string, error)                { return nil, nil }
func (*definitionDual) EnsureZoneExists(*models.DomainConfig) error { return nil }

type initializingObserver struct {
	noopConversionObserver
	calls int
}

func (o *initializingObserver) BeginToRC(string, any) ConversionSnapshot {
	o.calls++
	return nil
}

func TestRegisterFactoriesAndAuditing(t *testing.T) {
	isolateDefinitions(t)
	before := definitionInitializations
	Register[*definitionDual]("DUAL", Definition{FriendlyName: "Dual", Aliases: []string{"ALIAS"}})
	Register[*definitionDNS]("DNS", Definition{FriendlyName: "DNS"})
	Register[*None]("REG", Definition{FriendlyName: "Registrar"})

	for name, kind := range map[string]ProviderKind{"DUAL": KindDNS | KindRegistrar, "DNS": KindDNS, "REG": KindRegistrar} {
		def, ok := GetDefinition(name)
		if !ok || def.Kind != kind {
			t.Fatalf("%s: kind = %v, found = %v", name, def.Kind, ok)
		}
		if def.CanGetZones != (name == "DUAL") || def.DocCreateDomains != (name == "DUAL") {
			t.Fatalf("%s: unexpected zone capabilities", name)
		}
	}
	_ = AllDefinitions()
	if errs := AuditRecords("ALIAS", nil); len(errs) != 0 {
		t.Fatal(errs)
	}
	if definitionInitializations != before {
		t.Fatal("registration, accessors or auditing initialized a provider")
	}
	if errs := AuditRecords("ALIAS", models.Records{&models.RecordConfig{}}); len(errs) != 1 || errs[0].Error() != "record rejected" {
		t.Fatalf("audit errors = %v", errs)
	}

	observer := &initializingObserver{}
	overrideRole := func(o *CreateOptions) { o.RequestedRole = KindDNS | KindRegistrar }
	dns, err := CreateDNSProvider("ALIAS", map[string]string{"TYPE": "DUAL", "account": "one"}, json.RawMessage(`{"meta":true}`), overrideRole, WithConversionObserver(observer))
	if err != nil {
		t.Fatal(err)
	}
	reg, err := CreateRegistrar("DUAL", map[string]string{"TYPE": "ALIAS", "account": "one"}, overrideRole, WithConversionObserver(observer))
	if err != nil {
		t.Fatal(err)
	}
	another, err := CreateDNSProvider("DUAL", map[string]string{"account": "two"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, b, c := dns.(*definitionDual), reg.(*definitionDual), another.(*definitionDual)
	if a == b || a == c || b == c || a.account != "one" || c.account != "two" {
		t.Fatal("instances are not independent per account and role")
	}
	if a.options.RequestedRole != KindDNS || b.options.RequestedRole != KindRegistrar || string(a.meta) != `{"meta":true}` {
		t.Fatal("options or metadata were not forwarded correctly")
	}
	if a.options.ConversionObserver != observer || b.options.ConversionObserver != observer || observer.calls != 2 {
		t.Fatal("observer was not available during initialization")
	}
	if _, err := CreateDNSProvider("REG", nil, nil); err == nil {
		t.Fatal("registrar-only implementation accepted DNS role")
	}
	if _, err := CreateRegistrar("DNS", nil); err == nil {
		t.Fatal("DNS-only implementation accepted registrar role")
	}
	if _, err := CreateDNSProvider("DUAL", map[string]string{"TYPE": "DNS"}, nil); err == nil {
		t.Fatal("different provider types accepted as aliases")
	}
	for _, role := range []ProviderKind{KindDNS, KindRegistrar} {
		var result any
		if role == KindDNS {
			result, err = CreateDNSProvider("DUAL", map[string]string{"fail": "yes"}, nil)
		} else {
			result, err = CreateRegistrar("DUAL", map[string]string{"fail": "yes"})
		}
		if result != nil || err == nil || err.Error() != "initialization failed" {
			t.Fatalf("failed initializer exposed a partial instance: %v, %v", result, err)
		}
	}
	def, _ := GetDefinition("DUAL")
	for _, options := range []*CreateOptions{nil, {}, {ConversionObserver: nil}} {
		instance, err := def.Initializer(nil, nil, options)
		if err != nil {
			t.Fatal(err)
		}
		got := instance.(*definitionDual).options
		if got.RequestedRole != 0 || reflect.TypeOf(got.ConversionObserver) != reflect.TypeFor[noopConversionObserver]() {
			t.Fatalf("unexpected default options: %+v", got)
		}
	}
	if result, err := def.Initializer(nil, nil, &CreateOptions{RequestedRole: KindDNS | KindRegistrar}); result != nil || err == nil {
		t.Fatal("combined role accepted")
	}
}

func TestDefinitionPointersAndMetadata(t *testing.T) {
	isolateDefinitions(t)
	postWrites := 0
	input := Definition{
		FriendlyName: "Example", Aliases: []string{"ALIAS", "ZZZ_ALIAS"}, Maintainer: "@example", DefaultTTL: 300,
		DocsURL: "docs", PortalURL: "portal", Notes: "notes",
		CredFields: []CredsField{{Key: "key", Choices: []string{"one"}, ShowIf: map[string]string{"mode": "one"}, Validator: func(string) error { return nil }}},
		PostWrite:  func(map[string]string) error { postWrites++; return nil },
		Features:   DocumentationNotes{CanConcur: Can("legacy"), CanUseCAA: Can("CAA", "link"), CanGetZones: Can("enumerate"), DocCreateDomains: Can()},
		CanConcur:  Cannot("explicit"), CanAutoDNSSEC: Unimplemented("pending"),
		CanUseDSForChildren: Can(), DocDualHost: Cannot(), DocOfficiallySupported: Can(),
		RecordIdentity: func(*models.RecordConfig) string { return "identity" },
	}
	Register[*definitionDNS]("ZZZ", input)
	Register[*None]("AAA", Definition{FriendlyName: "First"})
	def, ok := GetDefinition("ZZZ")
	if !ok || def == nil {
		t.Fatal("canonical definition not found")
	}
	for _, name := range []string{"ZZZ", "ALIAS", "ZZZ_ALIAS"} {
		got, ok := GetDefinition(name)
		if !ok || got != def {
			t.Fatalf("%s: lookup did not return the same stored pointer", name)
		}
	}
	if missing, ok := GetDefinition("MISSING"); ok || missing != nil {
		t.Fatalf("unknown definition = %v, %v; want nil, false", missing, ok)
	}
	all := AllDefinitions()
	first, _ := GetDefinition("AAA")
	if len(all) != 2 || all[0] != first || all[1] != def {
		t.Fatalf("canonical ordering: %v", all)
	}
	if def.TypeName != "ZZZ" || def.ImplementationType != reflect.TypeFor[*definitionDNS]() ||
		def.Aliases[0] != "ALIAS" || def.CredFields[0].Choices[0] != "one" || def.CredFields[0].ShowIf["mode"] != "one" ||
		def.Features[CanUseCAA].Comment != "CAA" || def.DerivedFeatures[CanUseCAA].Link != "link" ||
		def.CanConcur.Comment != "explicit" || def.CanAutoDNSSEC.Comment != "pending" ||
		!def.CanUseDSForChildren.HasFeature || def.DocDualHost.HasFeature || !def.DocOfficiallySupported.HasFeature {
		t.Fatalf("registration did not retain supplied metadata: %+v", def)
	}
	if !input.Features[CanGetZones].HasFeature || !input.Features[DocCreateDomains].HasFeature ||
		!def.Features[CanGetZones].HasFeature || !def.Features[DocCreateDomains].HasFeature {
		t.Fatal("deriving interface capabilities overwrote the supplied feature notes")
	}
	if ProviderHasCapability("ALIAS", CanConcur) || !ProviderHasCapability("ALIAS", CanUseCAA) ||
		ProviderHasCapability("ALIAS", CanAutoDNSSEC) || !def.DerivedFeatures[CanAutoDNSSEC].Unimplemented ||
		ProviderHasCapability("ALIAS", CanGetZones) || ProviderHasCapability("ALIAS", DocCreateDomains) {
		t.Fatal("explicit or interface-derived capabilities did not override legacy features")
	}
	if GetDefaultTTL("ALIAS") != 300 || GetRecordIdentity("ALIAS")(&models.RecordConfig{}) != "identity" {
		t.Fatal("alias accessors lost metadata")
	}
	if def.FriendlyName != "Example" || def.Kind != KindDNS || def.DocsURL != "docs" || def.PortalURL != "portal" || def.Notes != "notes" || def.Maintainer != "@example" {
		t.Fatalf("definition metadata = %+v", def)
	}
	if err := def.PostWrite(nil); err != nil || postWrites != 1 {
		t.Fatal("PostWrite not preserved")
	}
	if CanonicalName("ALIAS") != "ZZZ" || CanonicalName("MISSING") != "MISSING" {
		t.Fatal("canonical name resolution failed")
	}
}

func TestDefinitionDocumentationURLs(t *testing.T) {
	for _, tc := range []struct {
		name, override, vendorURL, want string
	}{
		{"MiXeD_TYPE", "", "", "https://docs.dnscontrol.org/provider/mixed_type"},
		{"RENAMED", "https://docs.dnscontrol.org/provider/legacy", "https://api.example.test/docs", "https://docs.dnscontrol.org/provider/legacy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateDefinitions(t)
			Register[*None](tc.name, Definition{
				FriendlyName: "An unrelated brand name", Aliases: []string{"OLD_NAME"},
				DocsURL: tc.override, VendorAPIDocURL: tc.vendorURL,
			})
			for _, name := range []string{tc.name, "OLD_NAME"} {
				def, _ := GetDefinition(name)
				if def.DocsURL != tc.want || def.VendorAPIDocURL != tc.vendorURL {
					t.Fatalf("%s: documentation URLs = %q, %q; want %q, %q", name, def.DocsURL, def.VendorAPIDocURL, tc.want, tc.vendorURL)
				}
			}
		})
	}
}

type valueInitializer struct{}

func (valueInitializer) Initialize(map[string]string, json.RawMessage, *CreateOptions) error {
	return nil
}

func TestRegisterRejectsInvalidDefinitions(t *testing.T) {
	isolateDefinitions(t)
	valid := Definition{FriendlyName: "Example"}
	checks := []struct {
		name     string
		register func()
		message  string
	}{
		{"value", func() { Register[valueInitializer]("BAD", valid) }, "pointer to a concrete struct"},
		{"interface", func() { Register[InitializableProvider]("BAD", valid) }, "pointer to a concrete struct"},
		{"no roles", func() { Register[*valueInitializer]("BAD", valid) }, "neither DNS nor registrar"},
		{"no auditor", func() { Register[*missingAuditor]("BAD", valid) }, "neither DNS nor registrar"},
		{"no friendly name", func() { Register[*None]("BAD", Definition{}) }, "FriendlyName"},
		{"redundant documentation URL", func() {
			Register[*None]("MiXeD_TYPE", Definition{FriendlyName: "Bad", DocsURL: "https://docs.dnscontrol.org/provider/mixed_type"})
		}, "DocsURL override matches derived URL"},
		{"nil feature note", func() {
			Register[*None]("BAD", Definition{FriendlyName: "Bad", Features: DocumentationNotes{CanConcur: nil}})
		}, "must not be nil"},
		{"empty name", func() { Register[*None]("", valid) }, "invalid"},
		{"placeholder name", func() { Register[*None]("-", valid) }, "invalid"},
		{"derived field", func() { Register[*None]("BAD", Definition{FriendlyName: "Bad", Kind: KindDNS}) }, "derived"},
		{"duplicate alias", func() { Register[*None]("BAD", Definition{FriendlyName: "Bad", Aliases: []string{"ALIAS", "ALIAS"}}) }, "already registered"},
		{"self alias", func() { Register[*None]("BAD", Definition{FriendlyName: "Bad", Aliases: []string{"BAD"}}) }, "already registered"},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			assertRegistrationPanics(t, check.message, check.register)
			if len(definitions) != 0 {
				t.Fatal("invalid registration was partially published")
			}
		})
	}
}

type missingAuditor struct{}

func (*missingAuditor) GetNameservers(string) ([]*models.Nameserver, error) { return nil, nil }
func (*missingAuditor) GetZoneRecords(*models.DomainConfig) (models.Records, error) {
	return nil, nil
}
func (*missingAuditor) GetZoneRecordsCorrections(*models.DomainConfig, models.Records) ([]*models.Correction, int, error) {
	return nil, 0, nil
}

func (*missingAuditor) Initialize(map[string]string, json.RawMessage, *CreateOptions) error {
	return nil
}

func assertRegistrationPanics(t *testing.T, message string, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(r.(string), message) {
			t.Errorf("panic = %v, want containing %q", r, message)
		}
	}()
	f()
}

func TestRegistrationNameCollisions(t *testing.T) {
	for _, canonicalFirst := range []bool{false, true} {
		t.Run(fmt.Sprint("canonical first=", canonicalFirst), func(t *testing.T) {
			isolateDefinitions(t)
			canonical := func() { Register[*None]("ALIAS", Definition{FriendlyName: "Registrar"}) }
			alias := func() { Register[*definitionDNS]("DNS", Definition{FriendlyName: "DNS", Aliases: []string{"ALIAS"}}) }
			if canonicalFirst {
				canonical()
				assertRegistrationPanics(t, "already registered", alias)
			} else {
				alias()
				assertRegistrationPanics(t, "already registered", canonical)
			}
		})
	}

	t.Run("canonical and alias conflicts", func(t *testing.T) {
		isolateDefinitions(t)
		Register[*None]("FIRST", Definition{FriendlyName: "First", Aliases: []string{"ALIAS"}})
		for _, def := range []Definition{
			{FriendlyName: "Second", Aliases: []string{"FIRST"}},
			{FriendlyName: "Second", Aliases: []string{"ALIAS"}},
		} {
			assertRegistrationPanics(t, "already registered", func() { Register[*None]("SECOND", def) })
		}
		assertRegistrationPanics(t, "already registered", func() { Register[*None]("ALIAS", Definition{FriendlyName: "Second"}) })
		assertRegistrationPanics(t, "already registered", func() { Register[*None]("FIRST", Definition{FriendlyName: "Second"}) })
	})
}

func TestNoneIsOnlyARegistrar(t *testing.T) {
	def, ok := GetDefinition("NONE")
	if !ok || def.Kind != KindRegistrar || def.RecordAuditor != nil {
		t.Fatalf("NONE definition = %+v", def)
	}
	if _, ok := any(&None{}).(DNSServiceProvider); ok {
		t.Fatal("NONE advertises DNS support")
	}
	if _, err := CreateRegistrar("NONE", nil); err != nil {
		t.Fatal(err)
	}
}
