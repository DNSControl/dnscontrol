package providers_test

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	_ "github.com/DNSControl/dnscontrol/v5/pkg/providers/_all"
)

func TestAllProvidersUseDefinitions(t *testing.T) {
	for _, def := range providers.AllDefinitions() {
		if got, ok := providers.GetDefinition(def.TypeName); !ok || got != def {
			t.Fatalf("%s: canonical definition lookup failed", def.TypeName)
		}
		if def.SupportedTypes == nil || !def.UsesSupportedTypes() {
			t.Errorf("%s: missing exhaustive SupportedTypes declaration", def.TypeName)
		}
		for capability, note := range def.Features {
			switch capability {
			case providers.CanAutoDNSSEC, providers.CanConcur, providers.CanUseDSForChildren,
				providers.DocDualHost, providers.DocOfficiallySupported:
				t.Errorf("%s: %s belongs in its named field", def.TypeName, capability)
			}
			if note.Comment == "" && note.Link == "" {
				t.Errorf("%s: Features[%s] has no annotation to retain", def.TypeName, capability)
			}
		}
		if !def.Kind.Has(providers.KindDNS) {
			if len(def.SupportedTypes) != 0 {
				t.Errorf("%s: registrar-only provider declares record support", def.TypeName)
			}
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
