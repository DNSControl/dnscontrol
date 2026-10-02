package providers_test

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	_ "github.com/DNSControl/dnscontrol/v5/pkg/providers/_all"
)

func TestAllProvidersUseDefinitions(t *testing.T) {
	// During the consumer migration, every legacy entry must be a view of a
	// unified definition, never a provider registered only through the old API.
	roles := map[string]providers.ProviderKind{}
	for name := range providers.DNSProviderTypes {
		roles[name] |= providers.KindDNS
	}
	for name := range providers.RegistrarTypes {
		roles[name] |= providers.KindRegistrar
	}
	for name, kind := range roles {
		def, ok := providers.GetDefinition(name)
		if !ok {
			t.Errorf("%s has no unified definition", name)
			continue
		}
		if def.Kind != kind {
			t.Errorf("%s: definition roles = %v, legacy roles = %v", name, def.Kind, kind)
		}
	}
	for _, def := range providers.AllDefinitions() {
		if roles[def.TypeName] != def.Kind {
			t.Errorf("%s is missing from the legacy views", def.TypeName)
		}
		if !def.Kind.Has(providers.KindDNS) {
			continue
		}
		t.Run(def.TypeName, func(t *testing.T) {
			// Auditing must work before Initialize, without credentials, clients,
			// or caches. Exercise both an empty zone and an ordinary record.
			dc := &models.DomainConfig{Name: "example.com"}
			for _, records := range []models.Records{nil, {dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1")}} {
				if errs := providers.AuditRecords(def.TypeName, records); len(errs) != 0 {
					t.Errorf("credential-free audit: %v", errs)
				}
			}
		})
	}
}

func TestMigratedZoneCapabilities(t *testing.T) {
	// These old feature flags disagreed with the operations implemented by
	// the provider. Interface discovery must describe the usable operations.
	for _, name := range []string{"DNSCALE", "DYNU", "LINODE", "MYTHICBEASTS", "OPENWRT"} {
		if providers.ProviderHasCapability(name, providers.CanGetZones) {
			t.Errorf("%s advertises zone listing without ListZones", name)
		}
	}
	for name, want := range map[string]bool{
		"AUTODNS": true, "EXOSCALE": true, "AZURE_PRIVATE_DNS": false, "MIKROTIK": false,
	} {
		if got := providers.ProviderHasCapability(name, providers.DocCreateDomains); got != want {
			t.Errorf("%s: zone creation = %v, want %v", name, got, want)
		}
	}
}
