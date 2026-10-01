package cloudns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

type definitionObserver struct{ providers.ConversionObserver }

func TestDefinitionInitializesAccountsAndObservers(t *testing.T) {
	observer := &definitionObserver{(&providers.CreateOptions{}).WithDefaults().ConversionObserver}
	config := map[string]string{"auth-id": "one", "auth-password": "test-password"}
	dns, err := providers.CreateDNSProvider("CLOUDNS", config, nil, providers.WithConversionObserver(observer))
	if err != nil {
		t.Fatal(err)
	}
	reg, err := providers.CreateRegistrar("CLOUDNS", config, providers.WithConversionObserver(observer))
	if err != nil {
		t.Fatal(err)
	}
	another, err := providers.CreateDNSProvider("CLOUDNS", map[string]string{"sub-auth-id": "two", "auth-password": "test-password"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	d, r, a := dns.(*cloudnsProvider), reg.(*cloudnsProvider), another.(*cloudnsProvider)
	if d == r || d == a || r == a || d.requestLimit == r.requestLimit || d.requestLimit == a.requestLimit || r.requestLimit == a.requestLimit {
		t.Fatal("accounts or roles share an instance or rate limiter")
	}
	if d.creds.id != "one" || r.creds.id != "one" || a.creds.id != "" || a.creds.subid != "two" {
		t.Fatal("account credentials were not isolated")
	}
	if d.observer != observer || r.observer != observer || a.observer == nil || a.observer == observer {
		t.Fatal("initializer did not install the supplied or default observer")
	}
	if p, err := providers.CreateDNSProvider("CLOUDNS", nil, nil); p != nil || err == nil {
		t.Fatalf("missing credentials returned a partial provider: %v, %v", p, err)
	}
}
