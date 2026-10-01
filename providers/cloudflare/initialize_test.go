package cloudflare

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/cloudflare/cloudflare-go"
)

func TestRegisteredDefinition(t *testing.T) {
	def, ok := providers.GetDefinition("CLOUDFLAREAPI")
	if !ok || def.Kind != providers.KindDNS || def.FriendlyName != "Cloudflare" ||
		def.DocsURL != "https://docs.dnscontrol.org/provider/cloudflareapi" || !def.CanGetZones || !def.DocCreateDomains {
		t.Fatalf("Cloudflare definition = %+v", def)
	}
	if !reflect.DeepEqual(def.DerivedFeatures, def.Features) {
		t.Fatal("migration changed legacy capabilities")
	}
	if errors := providers.AuditRecords("CLOUDFLAREAPI", nil); len(errors) != 0 {
		t.Fatal(errors)
	}
	if _, err := providers.CreateRegistrar("CLOUDFLAREAPI", nil); err == nil {
		t.Fatal("Cloudflare advertised registrar support")
	}
}

func TestInitializeCredentialsAndMetadata(t *testing.T) {
	for _, config := range []map[string]string{
		{"apitoken": "test-token", "accountid": "account-one"},
		{"apikey": "test-key", "apiuser": "user@example.com", "accountid": "account-two"},
	} {
		meta := json.RawMessage(`{"manage_workers":true,"manage_single_redirects":true,"transcode_log":"example.log"}`)
		observer := &initializationObserver{ConversionObserver: (*providers.CreateOptions)(nil).WithDefaults().ConversionObserver}
		instance, err := providers.CreateDNSProvider("CLOUDFLAREAPI", config, meta, providers.WithConversionObserver(observer))
		if err != nil {
			t.Fatal(err)
		}
		p := instance.(*cloudflareProvider)
		if p.cfClient == nil || p.accountID != config["accountid"] || !p.manageWorkers || !p.manageSingleRedirects || p.tcLogFilename != "example.log" || p.tcLogFh != nil || p.observer != observer {
			t.Fatal("Cloudflare initialization lost credentials/metadata or opened the transcode log")
		}
		other, err := providers.CreateDNSProvider("CLOUDFLAREAPI", config, nil)
		if err != nil {
			t.Fatal(err)
		}
		q := other.(*cloudflareProvider)
		if p == q || p.cfClient == q.cfClient || q.manageWorkers || q.manageSingleRedirects {
			t.Fatal("Cloudflare accounts share mutable state or metadata")
		}
		for index, provider := range []*cloudflareProvider{p, q} {
			zoneName := fmt.Sprintf("zone%d.example.com", index)
			transport := initializationTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					return nil, fmt.Errorf("unexpected method: %s", r.Method)
				}
				var body string
				switch r.URL.Path {
				case "/zones":
					body = fmt.Sprintf(`{"success":true,"result":[{"id":"zone-id","name":%q}],"result_info":{"page":1,"total_pages":1}}`, zoneName)
				case "/zones/zone-id/dns_records":
					if r.URL.Query().Get("page") != "1" {
						return nil, fmt.Errorf("unexpected record page: %s", r.URL)
					}
					body = fmt.Sprintf(`{"success":true,"result":[{"id":"record-id","name":%q,"type":"A","content":"192.0.2.1","ttl":300}],"result_info":{"page":1,"total_pages":1}}`, zoneName)
				default:
					return nil, fmt.Errorf("unexpected path: %s", r.URL.Path)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			if err := cloudflare.HTTPClient(&http.Client{Transport: transport})(provider.cfClient); err != nil {
				t.Fatal(err)
			}
			if err := cloudflare.BaseURL("https://cloudflare.invalid")(provider.cfClient); err != nil {
				t.Fatal(err)
			}
			if err := cloudflare.UsingRetryPolicy(0, 0, 0)(provider.cfClient); err != nil {
				t.Fatal(err)
			}
			if zones, err := provider.ListZones(); err != nil || !reflect.DeepEqual(zones, []string{zoneName}) {
				t.Fatalf("instance zone cache: %v, %v", zones, err)
			}
			if _, err := provider.getRecordsForDomain("zone-id", models.MustNewDomainConfig(zoneName)); err != nil {
				t.Fatal(err)
			}
		}
		if observer.begins != 1 || observer.ends != 1 {
			t.Fatalf("conversion observation lost: begins=%d, ends=%d", observer.begins, observer.ends)
		}
	}
	for _, test := range []struct {
		config map[string]string
		meta   json.RawMessage
		want   string
	}{
		{nil, nil, "apikey and apiuser must be provided"},
		{map[string]string{"apikey": "test"}, nil, "apikey and apiuser must be provided"},
		{map[string]string{"apitoken": "test", "apiuser": "test"}, nil, "apikey and apiuser should not be provided"},
		{map[string]string{"apitoken": "test"}, json.RawMessage(`{broken`), "invalid character"},
	} {
		instance, err := providers.CreateDNSProvider("CLOUDFLAREAPI", test.config, test.meta)
		if instance != nil || err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("invalid configuration returned %v, %v; want %s", instance, err, test.want)
		}
	}
}

type initializationTransport func(*http.Request) (*http.Response, error)

func (f initializationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type initializationObserver struct {
	providers.ConversionObserver
	begins, ends int
}

func (o *initializationObserver) BeginToRC(string, any) providers.ConversionSnapshot {
	o.begins++
	return nil
}

func (o *initializationObserver) EndToRC(string, providers.ConversionSnapshot, any, models.Records, error) {
	o.ends++
}
