package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/stretchr/testify/require"
)

func TestDomainProviderMetadataClientReuse(t *testing.T) {
	for _, initializer := range syntaxInitializers {
		t.Run(initializer.name, func(t *testing.T) {
			registrars, dns := registerSyntaxTestProvider(t)
			cfg := &models.DNSConfig{
				Registrars: []*models.RegistrarConfig{{Name: "primary", Type: "-"}},
				DNSProviders: []*models.DNSProviderConfig{
					{Name: "primary", Type: "-"},
					{Name: "secondary", Type: "-"},
				},
			}
			uses := []struct {
				entry, metadata string
				client          int
			}{
				{"primary", `{"a":1,"b":2}`, 0},
				{"primary", `{"a":1,"b":2}`, 0}, // Same bytes, different domain.
				{"primary", `{"a":2,"b":2}`, 1},
				{"primary", `{"b":2,"a":1}`, 2},   // Equivalent JSON, different bytes.
				{"secondary", `{"a":1,"b":2}`, 3}, // Same metadata, different credentials.
				{"primary", "", 4},
				{"primary", "", 4},
				{"primary", "null", 5}, // Explicit null differs from absent metadata.
				{"primary", `[1,"two",false]`, 6},
				{"primary", `"text"`, 7},
			}
			for i, use := range uses {
				d := models.MustNewDomainConfig(fmt.Sprintf("example%d.com", i))
				d.RegistrarName = "primary"
				d.DNSProviderNames = map[string]int{use.entry: -1}
				if use.metadata != "" {
					d.DNSProviderMetadata = map[string]json.RawMessage{use.entry: json.RawMessage(use.metadata)}
				}
				cfg.Domains = append(cfg.Domains, d)
			}
			d := models.MustNewDomainConfig("both.example.com")
			d.RegistrarName = "primary"
			d.DNSProviderNames = map[string]int{"primary": -1, "secondary": 0}
			d.DNSProviderMetadata = map[string]json.RawMessage{
				"primary":   json.RawMessage(`{"a":2,"b":2}`),
				"secondary": json.RawMessage(`{"a":1,"b":2}`),
			}
			cfg.Domains = append(cfg.Domains, d)
			_, err := preloadProviders(cfg)
			require.NoError(t, err)
			creds := map[string]map[string]string{
				"primary":   {"TYPE": providerSyntaxTestType, "account": "one"},
				"secondary": {"TYPE": providerSyntaxTestType, "account": "two", "_exclude_from_defaults": "true"},
			}
			require.NoError(t, initializer.init(cfg, creds))
			require.Len(t, *registrars, 1)
			require.Empty(t, (*registrars)[0].metadata, "registrar initialization receives no configMetadata")
			require.Len(t, *dns, 8)
			for i, use := range uses {
				instance := cfg.Domains[i].DNSProviderInstances[0]
				require.Same(t, (*dns)[use.client], instance.Driver)
				require.Equal(t, use.metadata, string((*dns)[use.client].metadata))
				require.Equal(t, creds[use.entry]["account"], (*dns)[use.client].account)
				require.Equal(t, use.entry == "primary", instance.IsDefault)
				require.NotSame(t, cfg.Domains[i].RegistrarInstance.Driver, instance.Driver)
			}
			require.Same(t, (*dns)[1], d.DNSProviderInstances[0].Driver)
			require.Same(t, (*dns)[3], d.DNSProviderInstances[1].Driver)
		})
	}
}

func TestDomainProviderMetadataValidation(t *testing.T) {
	for _, tt := range []struct {
		name, legacy, metadata, wantError string
		unused, overwritten               bool
	}{
		{name: "legacy fallback", legacy: `{"setting":"legacy"}`},
		{name: "same duplicate", legacy: `{"setting":"same"}`, metadata: `{"setting":"same"}`, wantError: "duplicate configMetadata error"},
		{name: "different duplicate", legacy: `{"setting":"legacy"}`, metadata: `{"setting":"domain"}`, wantError: "duplicate configMetadata error"},
		{name: "null duplicate", legacy: "null", metadata: "null", wantError: "duplicate configMetadata error"},
		{name: "overwritten legacy declaration", legacy: `{"setting":"legacy"}`, metadata: `{}`, overwritten: true, wantError: "duplicate configMetadata error"},
		{name: "unused metadata", metadata: `{}`, unused: true, wantError: "unused DNS provider"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := models.MustNewDomainConfig("example.com")
			d.RegistrarName = "none"
			d.DNSProviderNames = map[string]int{"account": -1}
			if tt.metadata != "" {
				name := "account"
				if tt.unused {
					name = "unused"
				}
				d.DNSProviderMetadata = map[string]json.RawMessage{name: json.RawMessage(tt.metadata)}
			}
			cfg := &models.DNSConfig{
				Registrars:   []*models.RegistrarConfig{{Name: "none", Type: "NONE"}},
				DNSProviders: []*models.DNSProviderConfig{{Name: "account", Type: providerSyntaxTestType, Metadata: json.RawMessage(tt.legacy)}},
				Domains:      []*models.DomainConfig{d},
			}
			if tt.overwritten {
				cfg.DNSProviders = append(cfg.DNSProviders, &models.DNSProviderConfig{Name: "account", Type: providerSyntaxTestType})
			}
			_, err := preloadProviders(cfg)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				if !tt.unused {
					require.EqualError(t, err, `duplicate configMetadata error: domain "example.com" defines configMetadata for "account" in both NewDnsProvider() and SERVICE()`)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.legacy, string(d.DNSProviderInstances[0].Metadata))
			}
		})
	}
}

func TestDomainProviderMetadataIRLoading(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dnsconfig.json")
	data := `{
		"registrars": [{"name":"none","type":"NONE"}],
		"dns_providers": [{"name":"account","type":"TEST_PROVIDER_SYNTAX"}],
		"domains": [{"name":"example.com","registrar":"none","dnsProviders":{"account":0},
			"dnsProviderMetadata":{"account":{"nested":[1,"two",null]}}}]
	}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
	cfg, err := GetDNSConfig(GetDNSConfigArgs{JSONFile: path})
	require.NoError(t, err)
	want := `{"nested":[1,"two",null]}`
	require.Equal(t, want, string(cfg.Domains[0].DNSProviderInstances[0].Metadata))
	encoded, err := json.Marshal(cfg)
	require.NoError(t, err)
	var roundTrip models.DNSConfig
	require.NoError(t, json.Unmarshal(encoded, &roundTrip))
	require.Equal(t, want, string(roundTrip.Domains[0].DNSProviderMetadata["account"]))
}
