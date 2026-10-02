package providers_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/normalize"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func providerDomain(name string) *models.DomainConfig {
	dc := models.MustNewDomainConfig("example.com")
	dc.RegistrarName = "NONE"
	dc.DNSProviderInstances = []*models.DNSProviderInstance{{Name: "dns", ProviderType: name}}
	return dc
}

func TestDeclaredBaselinePassesAuditing(t *testing.T) {
	// Check declarations against the actual credential-free auditors, including
	// their placement and data restrictions, rather than only the old flags.
	for _, def := range providers.AllDefinitions() {
		if !def.Kind.Has(providers.KindDNS) {
			continue
		}
		for _, record := range []struct{ label, rtype, data string }{
			{"www", "A", "192.0.2.1"},
			{"www", "AAAA", "2001:db8::1"},
			{"@", "CAA", `0 issue "ca.target.test"`},
			{"www", "CNAME", "target.target.test."},
			{"@", "MX", "10 mail.target.test."},
			{"child", "NS", "ns.target.test."},
			{"_sip._tcp", "SRV", "0 5 5060 sip.target.test."},
			{"@", "TXT", `"text"`},
		} {
			if !def.RecordTypeSupport(record.rtype).HasFeature {
				continue
			}
			t.Run(def.TypeName+"/"+record.rtype, func(t *testing.T) {
				dc := providerDomain(def.TypeName)
				dc.AddRecordConfig(dc.MustNewRecordConfigParse(record.label, 3600, record.rtype, record.data))
				if errs := normalize.ValidateAndNormalizeConfig(&models.DNSConfig{Domains: []*models.DomainConfig{dc}}); len(errs) != 0 {
					t.Fatal(errs)
				}
			})
		}
	}
}

func TestBaselineExceptions(t *testing.T) {
	// These providers reject or discard types that legacy validation implicitly
	// allowed. NS here means a child delegation, not injected apex nameservers.
	for name, types := range map[string]string{
		"ADGUARDHOME": "MX NS TXT", "FORTIGATE": "MX NS TXT", "NETBIRD": "MX NS TXT",
		"OPENWRT": "NS TXT", "UNIFI": "NS", "AZURE_PRIVATE_DNS": "NS",
		"EXOSCALE": "NS", "NETCUP": "NS", "INFOBLOX": "NS", "MITTWALD": "NS",
		"WEBSUPPORT": "NS", "OPENPROVIDER": "NS",
	} {
		for rtype := range strings.FieldsSeq(types) {
			t.Run(name+"/"+rtype, func(t *testing.T) {
				dc := providerDomain(name)
				data := map[string]string{"MX": "10 mail.target.test.", "NS": "ns.target.test.", "TXT": `"text"`}[rtype]
				dc.AddRecordConfig(dc.MustNewRecordConfigParse("child", 3600, rtype, data))
				errs := normalize.ValidateAndNormalizeConfig(&models.DNSConfig{Domains: []*models.DomainConfig{dc}})
				if !strings.Contains(fmt.Sprint(errs), "uses "+rtype+" records, but DNS provider type "+name+" does not support") {
					t.Fatalf("missing exhaustive type check: %v", errs)
				}
			})
		}
	}
}

func TestSharedAndProprietaryTypes(t *testing.T) {
	for _, tc := range []struct {
		provider, rtype, data string
		allowed               bool
	}{
		{"NAMECHEAP", "URL", "https://target.test/", true},
		{"PORKBUN", "URL", "https://target.test/", true},
		{"NAMECHEAP", "URL301", "https://target.test/", true},
		{"PORKBUN", "URL301", "https://target.test/", true},
		{"ROUTE53", "URL", "https://target.test/", false},
		{"ROUTE53", "R53_ALIAS", "A target.target.test. false Z123", true},
		{"CLOUDFLAREAPI", "R53_ALIAS", "A target.target.test. false Z123", false},
		{"CLOUDFLAREAPI", "CF_WORKER_ROUTE", `"example.com/*" "worker"`, true},
		{"ROUTE53", "CF_WORKER_ROUTE", `"example.com/*" "worker"`, false},
		{"NETLIFY", "NETLIFYV6", "", true},
		{"DNSIMPLE", "NETLIFYV6", "", false},
		{"POWERDNS", "LOC", "37 47 0 N 122 23 0 W 10m", false},
	} {
		t.Run(tc.provider+"/"+tc.rtype, func(t *testing.T) {
			dc := providerDomain(tc.provider)
			if tc.rtype == "NETLIFYV6" {
				// This pseudo-type has no RDATA arguments, matching its JS builder.
				dc.AddRecordConfig(dc.MustNewRecordConfig("www", 3600, tc.rtype))
			} else {
				dc.AddRecordConfig(dc.MustNewRecordConfigParse("www", 3600, tc.rtype, tc.data))
			}
			errs := normalize.ValidateAndNormalizeConfig(&models.DNSConfig{Domains: []*models.DomainConfig{dc}})
			if (len(errs) == 0) != tc.allowed {
				t.Fatalf("allowed=%v, errors=%v", tc.allowed, errs)
			}
			if !tc.allowed && !strings.Contains(fmt.Sprint(errs), "does not support") {
				t.Fatalf("missing exhaustive type check: %v", errs)
			}
		})
	}
}

func TestMigratedProvidersImportRecords(t *testing.T) {
	for _, def := range providers.AllDefinitions() {
		if !def.Kind.Has(providers.KindDNS) {
			continue
		}
		t.Run(def.TypeName, func(t *testing.T) {
			source := models.MustNewDomainConfig("source.example")
			source.AddRecordConfig(source.MustNewRecordConfig("www", 3600, "A", "192.0.2.1"))
			dest := providerDomain(def.TypeName)
			transform := "0.0.0.0~255.255.255.255~~192.0.2.2"
			r := dest.MustNewRecordConfig("@", 3600, privatetypes.TypeIMPORTTRANSFORM, transform, 3600, source.Name, source.Name)
			r.Metadata["transform_table"] = transform
			dest.AddRecordConfig(r)
			config := &models.DNSConfig{Domains: []*models.DomainConfig{source, dest}}
			if err := config.PostProcess(); err != nil {
				t.Fatal(err)
			}
			if errs := normalize.ValidateAndNormalizeConfig(config); len(errs) != 0 {
				t.Fatal(errs)
			}
			if len(dest.Records) != 1 || dest.Records[0].Type != "A" || dest.Records[0].GetLabel() != "www" || dest.Records[0].GetTargetIP().String() != "192.0.2.2" {
				t.Fatalf("unexpected transformed records: %v", dest.Records)
			}
		})
	}
}
