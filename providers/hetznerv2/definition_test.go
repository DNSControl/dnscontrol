package hetznerv2

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type definitionTransport func(*http.Request) (*http.Response, error)

func (f definitionTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestDefinitionBindsZoneCacheToRuntimeInstance(t *testing.T) {
	var instances []*hetznerv2Provider
	for _, account := range []string{"one", "two"} {
		dns, err := providers.CreateDNSProvider("HETZNER_V2", map[string]string{"api_token": account}, nil)
		if err != nil {
			t.Fatal(err)
		}
		p := dns.(*hetznerv2Provider)
		instances = append(instances, p)

		// If Initialize copies a constructed provider, its cache callback may
		// retain that discarded receiver. Block its original client and replace
		// the runtime receiver's client to verify which one the callback uses.
		hcloud.WithHTTPClient(&http.Client{Transport: definitionTransport(func(*http.Request) (*http.Response, error) {
			t.Error("zone cache used a discarded receiver's client")
			return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":{"code":"invalid_input","message":"discarded client"}}`)), Header: make(http.Header)}, nil
		})})(p.client)
		calls := 0
		p.client = hcloud.NewClient(hcloud.WithHTTPClient(&http.Client{Transport: definitionTransport(func(*http.Request) (*http.Response, error) {
			calls++
			body := fmt.Sprintf(`{"zones":[{"id":1,"name":"%s.example"}],"meta":{"pagination":{"next_page":null}}}`, account)
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}))
		for range 2 {
			zones, err := p.ListZones()
			if err != nil || !reflect.DeepEqual(zones, []string{account + ".example"}) {
				t.Fatalf("%s: zones = %v, error = %v", account, zones, err)
			}
		}
		if calls != 1 {
			t.Fatalf("%s: zone list fetched %d times, want one cached fetch", account, calls)
		}
	}
	if instances[0] == instances[1] || instances[0].client == instances[1].client {
		t.Fatal("accounts share a runtime instance or client")
	}
}
