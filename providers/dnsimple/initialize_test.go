package dnsimple

import (
	"reflect"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func TestRegisteredDefinition(t *testing.T) {
	def, ok := providers.GetDefinition("DNSIMPLE")
	if !ok || def.Kind != providers.KindDNS|providers.KindRegistrar || !def.CanGetZones || def.DocCreateDomains {
		t.Fatalf("DNSimple definition = %+v", def)
	}
	if !reflect.DeepEqual(def.DerivedFeatures, features) {
		t.Fatal("migration changed legacy capabilities")
	}
	if errors := providers.AuditRecords("DNSIMPLE", nil); len(errors) != 0 {
		t.Fatal(errors)
	}
}

func TestInitializeSeparateRolesAndAccounts(t *testing.T) {
	var instances []*dnsimpleProvider
	for _, token := range []string{"one", "two"} {
		config := map[string]string{"token": token, "baseurl": "https://example.invalid/"}
		dns, err := providers.CreateDNSProvider("DNSIMPLE", config, nil)
		if err != nil {
			t.Fatal(err)
		}
		reg, err := providers.CreateRegistrar("DNSIMPLE", config)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []*dnsimpleProvider{dns.(*dnsimpleProvider), reg.(*dnsimpleProvider)} {
			if p.AccountToken != token || p.BaseURL != config["baseurl"] || p.accountID != "" {
				t.Fatal("initialization lost config or fetched an account")
			}
			instances = append(instances, p)
		}
	}
	for i, p := range instances {
		p.onceFetchAccountID.Do(func() { p.accountID = "cached" })
		for _, other := range instances[i+1:] {
			if p == other || other.accountID != "" {
				t.Fatal("accounts or roles share their account cache")
			}
		}
	}
	if p, err := providers.CreateDNSProvider("DNSIMPLE", nil, nil); p != nil || err == nil || err.Error() != "missing DNSimple token" {
		t.Fatalf("DNS accepted missing token: %v, %v", p, err)
	}
	if p, err := providers.CreateRegistrar("DNSIMPLE", nil); p != nil || err == nil || err.Error() != "missing DNSimple token" {
		t.Fatalf("registrar accepted missing token: %v, %v", p, err)
	}
}
