package route53

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func TestRegisteredDefinition(t *testing.T) {
	def, ok := providers.GetDefinition("ROUTE53")
	if !ok || def.Kind != providers.KindDNS|providers.KindRegistrar || !def.CanGetZones || !def.DocCreateDomains {
		t.Fatalf("Route 53 definition = %+v", def)
	}
	for capability, note := range def.Features {
		if !reflect.DeepEqual(def.DerivedFeatures[capability], note) {
			t.Fatalf("migration changed annotation for %s", capability)
		}
	}
	if errors := providers.AuditRecords("ROUTE53", nil); len(errors) != 0 {
		t.Fatal(errors)
	}
}

// Exercise the real shared setup against a local endpoint. No AWS credentials,
// local profiles, or live service calls are needed.
func TestInitializeRequestedRole(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/2013-04-01/hostedzone" {
			t.Errorf("unexpected AWS request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, `<ListHostedZonesResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><HostedZones><HostedZone><Id>/hostedzone/TEST</Id><Name>example.com.</Name><CallerReference>test</CallerReference></HostedZone></HostedZones><IsTruncated>false</IsTruncated><MaxItems>100</MaxItems></ListHostedZonesResponse>`)
	}))
	defer server.Close()
	for key, value := range map[string]string{
		"AWS_ENDPOINT_URL":                    server.URL,
		"AWS_ENDPOINT_URL_ROUTE_53":           server.URL,
		"AWS_IGNORE_CONFIGURED_ENDPOINT_URLS": "false",
		"AWS_CONFIG_FILE":                     filepath.Join(t.TempDir(), "no-config"),
		"AWS_SHARED_CREDENTIALS_FILE":         filepath.Join(t.TempDir(), "no-credentials"),
		"AWS_EC2_METADATA_DISABLED":           "true", "AWS_PROFILE": "",
	} {
		t.Setenv(key, value)
	}
	var instances []*route53Provider
	observer := &initializationObserver{}
	for _, region := range []string{"", "us-east-1", "eu-west-1", "eusc-de-east-1"} {
		for _, role := range []providers.ProviderKind{providers.KindDNS, providers.KindRegistrar} {
			t.Run(fmt.Sprintf("%s/role-%d", region, role), func(t *testing.T) {
				config := map[string]string{"TYPE": "ROUTE53", "KeyId": "test", "SecretKey": "test", "Region": region, "DelegationSet": "test-set"}
				before := requests.Load()
				// Even an attempted override must not bypass the registrar check.
				opts := []providers.CreateOption{
					func(o *providers.CreateOptions) { o.RequestedRole = providers.KindDNS | providers.KindRegistrar },
					providers.WithConversionObserver(observer),
				}
				var instance any
				var err error
				if role == providers.KindDNS {
					instance, err = providers.CreateDNSProvider("ROUTE53", config, nil, opts...)
				} else {
					instance, err = providers.CreateRegistrar("ROUTE53", config, opts...)
				}
				if role == providers.KindRegistrar && region != "" && region != "us-east-1" {
					if err == nil || !strings.Contains(err.Error(), "only supported on the global AWS region us-east-1") || instance != nil {
						t.Fatalf("registrar accepted restricted region: %v, %v", instance, err)
					}
					if requests.Load() != before {
						t.Fatal("region rejection made an AWS request")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				p := instance.(*route53Provider)
				if requests.Load() != before+1 || p.observer != observer || p.client == nil || p.registrar == nil || *p.delegationSet != "test-set" {
					t.Fatal("shared AWS setup or observer injection was lost")
				}
				if zone, ok := p.getZoneByDomain("example.com"); !ok || *zone.Id != "/hostedzone/TEST" {
					t.Fatal("shared setup did not populate the zone cache")
				}
				instances = append(instances, p)
			})
		}
	}
	for i, instance := range instances {
		delete(instance.zonesByDomain, "example.com")
		for _, other := range instances[i+1:] {
			if instance == other || other.zonesByDomain["example.com"].Id == nil {
				t.Fatal("runtime instances share a cache")
			}
		}
	}
}

func TestInitializeRetainsProfileConflict(t *testing.T) {
	for _, role := range []providers.ProviderKind{providers.KindDNS, providers.KindRegistrar} {
		p := &route53Provider{}
		err := p.Initialize(map[string]string{"KeyId": "test", "Profile": "test"}, nil, &providers.CreateOptions{RequestedRole: role})
		if err == nil || err.Error() != "route53: cannot set both Profile and KeyId/SecretKey" || p.client != nil {
			t.Fatalf("profile conflict for role %d: %v", role, err)
		}
	}
}

type initializationObserver struct{}

func (*initializationObserver) BeginToRC(string, any) providers.ConversionSnapshot { return nil }
func (*initializationObserver) EndToRC(string, providers.ConversionSnapshot, any, models.Records, error) {
}
func (*initializationObserver) BeginToNative(string, models.Records) providers.ConversionSnapshot {
	return nil
}
func (*initializationObserver) EndToNative(string, providers.ConversionSnapshot, models.Records, any, error) {
}
