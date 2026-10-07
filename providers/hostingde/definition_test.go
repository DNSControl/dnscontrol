package hostingde

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func TestDefinitionInitializesSeparateRolesAndMetadata(t *testing.T) {
	config := map[string]string{"authToken": "test-token", "ownerAccountId": "owner", "filterAccountId": "filter", "baseURL": "https://example.invalid/"}
	dns, err := providers.CreateDNSProvider("HOSTINGDE", config, json.RawMessage(`{"default_ns":["ns.example.com"]}`))
	if err != nil {
		t.Fatal(err)
	}
	reg, err := providers.CreateRegistrar("HOSTINGDE", config)
	if err != nil {
		t.Fatal(err)
	}
	d, r := dns.(*hostingdeProvider), reg.(*hostingdeProvider)
	if d == r {
		t.Fatal("DNS and registrar roles share an instance")
	}
	if !reflect.DeepEqual(d.nameservers, []string{"ns.example.com"}) || !reflect.DeepEqual(r.nameservers, defaultNameservers) {
		t.Fatalf("DNS metadata leaked into registrar defaults: DNS=%v, registrar=%v", d.nameservers, r.nameservers)
	}
	for _, p := range []*hostingdeProvider{d, r} {
		if p.authToken != "test-token" || p.ownerAccountID != "owner" || p.filterAccountID != "filter" || p.baseURL != "https://example.invalid" {
			t.Fatal("initializer did not retain credentials or normalize the base URL")
		}
	}
	if p, err := providers.CreateDNSProvider("HOSTINGDE", config, json.RawMessage(`{`)); p != nil || err == nil {
		t.Fatalf("invalid metadata returned a partial provider: %v, %v", p, err)
	}
	if p, err := providers.CreateRegistrar("HOSTINGDE", nil); p != nil || err == nil {
		t.Fatalf("missing credentials returned a partial provider: %v, %v", p, err)
	}
}
