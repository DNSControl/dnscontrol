package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/credsfile"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/js"
	"github.com/DNSControl/dnscontrol/v5/pkg/normalize"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/stretchr/testify/require"
)

const providerSyntaxTestType = "TEST_PROVIDER_SYNTAX"

// Both roles use a deterministic local provider; corrections are never executed.
type syntaxTestProvider struct {
	account   string
	metadata  json.RawMessage
	nsLookups int
}

func (*syntaxTestProvider) AuditRecords(models.Records) []error { return nil }

func (p *syntaxTestProvider) GetNameservers(string) ([]*models.Nameserver, error) {
	p.nsLookups++
	return models.ToNameservers([]string{"ns1.example.org", "ns2.example.org", "ns3.example.org"})
}

func (*syntaxTestProvider) GetZoneRecords(*models.DomainConfig) (models.Records, error) {
	return nil, nil
}

func (*syntaxTestProvider) ListZones() ([]string, error) { return []string{"example.com"}, nil }

func (p *syntaxTestProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	names := make([]string, 0, len(dc.Nameservers))
	for _, ns := range dc.Nameservers {
		names = append(names, ns.Name)
	}
	return []*models.Correction{{Msg: fmt.Sprintf("delegate %s (%s) to %s", dc.Name, p.account, strings.Join(names, ","))}}, nil
}

func (*syntaxTestProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	changes, count, err := diff2.ByRecord(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}
	var corrections []*models.Correction
	for _, change := range changes {
		corrections = append(corrections, change.CreateCorrection(func() error {
			panic("provider syntax tests must not apply corrections")
		}))
	}
	return corrections, count, nil
}

var syntaxRegInstances, syntaxDNSInstances []*syntaxTestProvider

func init() {
	providers.Register[*syntaxTestProvider](providerSyntaxTestType, providers.Definition{
		FriendlyName: "Syntax test",
		Aliases:      []string{"TEST_PROVIDER_SYNTAX_ALIAS"},
	})
}

func (p *syntaxTestProvider) Initialize(config map[string]string, meta json.RawMessage, options *providers.CreateOptions) error {
	p.account, p.metadata = config["account"], meta
	if options.RequestedRole == providers.KindRegistrar {
		syntaxRegInstances = append(syntaxRegInstances, p)
	} else {
		syntaxDNSInstances = append(syntaxDNSInstances, p)
	}
	return nil
}

func registerSyntaxTestProvider(t *testing.T) (registrars, dns *[]*syntaxTestProvider) {
	t.Helper()
	syntaxRegInstances, syntaxDNSInstances = nil, nil
	t.Cleanup(func() { syntaxRegInstances, syntaxDNSInstances = nil, nil })
	return &syntaxRegInstances, &syntaxDNSInstances
}

var syntaxInitializers = []struct {
	name string
	init func(*models.DNSConfig, map[string]map[string]string) error
}{
	{"InitializeProviders", func(cfg *models.DNSConfig, creds map[string]map[string]string) error {
		_, err := InitializeProviders(cfg, creds, false)
		return err
	}},
	{"PInitializeProviders", func(cfg *models.DNSConfig, creds map[string]map[string]string) error {
		_, err := PInitializeProviders(cfg, creds, false)
		return err
	}},
}

func loadSyntaxConfig(t *testing.T, script string) *models.DNSConfig {
	t.Helper()
	cfg, err := js.ExecuteJavascriptString([]byte(script), false, nil)
	require.NoError(t, err)
	cfg, err = preloadProviders(cfg)
	require.NoError(t, err)
	for _, domain := range cfg.Domains {
		for _, record := range domain.Records {
			record.FilePos = "" // Syntax changes source positions, not DNS content.
		}
	}
	return cfg
}

func TestProviderSyntaxCorrectionPlans(t *testing.T) {
	scripts := []string{
		`NewRegistrar("primary", {setting: "dns"});
		NewDnsProvider("primary", {setting: "dns"}); NewDnsProvider("secondary", {setting: "other"});
		D("example.com", "primary", DnsProvider("primary", 2), DnsProvider("secondary", 0), A("@", "192.0.2.1"));
		D("example.net", "primary", DnsProvider("primary"), A("@", "192.0.2.2"));`,
		`D("example.com", REGISTRAR("primary"), SERVICE("primary", 2, {setting: "dns"}), SERVICE("secondary", 0, {setting: "other"}), A("@", "192.0.2.1"));
		D("example.net", REGISTRAR("primary"), SERVICE("primary", ALL_NS, {setting: "dns"}), A("@", "192.0.2.2"));`,
	}
	creds := map[string]map[string]string{
		"primary":   {"TYPE": providerSyntaxTestType, "account": "one"},
		"secondary": {"TYPE": providerSyntaxTestType, "account": "two", "_exclude_from_defaults": "true"},
	}
	for _, initializer := range syntaxInitializers {
		t.Run(initializer.name, func(t *testing.T) {
			var expectedPlan []string
			for i, script := range scripts {
				t.Run([]string{"legacy", "modern"}[i], func(t *testing.T) {
					registrars, dns := registerSyntaxTestProvider(t)
					cfg := loadSyntaxConfig(t, script)
					require.NoError(t, initializer.init(cfg, creds))
					require.Len(t, *registrars, 1)
					require.Len(t, *dns, 2)
					require.NotSame(t, (*registrars)[0], (*dns)[0])
					require.NotSame(t, (*dns)[0], (*dns)[1])
					require.Equal(t, "one", (*dns)[0].account)
					require.Equal(t, "two", (*dns)[1].account)
					require.JSONEq(t, `{"setting":"dns"}`, string((*dns)[0].metadata))
					require.JSONEq(t, `{"setting":"other"}`, string((*dns)[1].metadata))
					require.Same(t, cfg.Domains[0].RegistrarInstance.Driver, cfg.Domains[1].RegistrarInstance.Driver)
					require.Same(t, cfg.Domains[0].DNSProviderInstances[0].Driver, cfg.Domains[1].DNSProviderInstances[0].Driver)
					require.False(t, cfg.Domains[0].DNSProviderInstances[1].IsDefault)
					require.Empty(t, normalize.ValidateAndNormalizeConfig(cfg))

					var plan []string
					for _, domain := range cfg.Domains {
						delegation, _, err := generateDelegationCorrections(domain, domain.DNSProviderInstances, domain.RegistrarInstance)
						require.NoError(t, err)
						for _, c := range delegation {
							plan = append(plan, c.Msg)
						}
						for _, provider := range domain.DNSProviderInstances {
							corrections, _, _, err := generateZoneCorrections(domain, provider)
							require.NoError(t, err)
							for _, c := range corrections {
								plan = append(plan, domain.Name+" / "+provider.Name+": "+c.Msg)
							}
						}
					}
					require.NotEmpty(t, plan)
					require.Contains(t, plan, "delegate example.com (one) to ns1.example.org,ns2.example.org")
					require.Equal(t, 0, (*dns)[1].nsLookups, "nsCount=0 must not fetch nameservers")
					if i == 0 {
						expectedPlan = plan
					} else {
						require.Equal(t, expectedPlan, plan)
					}
				})
			}
		})
	}
}

func TestProviderSyntaxCredentialResolution(t *testing.T) {
	tests := []struct {
		name, script string
		creds        map[string]map[string]string
		wantError    string
		wantReg      int
		wantDNS      int
	}{
		{"DNS only", `D("example.com", REGISTRAR("none"), SERVICE("account", 0));`,
			map[string]map[string]string{"none": {"TYPE": "NONE"}, "account": {"TYPE": providerSyntaxTestType}}, "", 0, 1},
		{"registrar only", `D("example.com", REGISTRAR("account"));`,
			map[string]map[string]string{"account": {"TYPE": providerSyntaxTestType}}, "", 1, 0},
		{"unused account without credentials", `DEFAULTS(REGISTRAR("unused"), SERVICE("unused"));`, nil, "", 0, 0},
		{"missing entry", `D("example.com", REGISTRAR("account"));`, nil, "missing", 0, 0},
		{"missing TYPE", `D("example.com", REGISTRAR("account"));`,
			map[string]map[string]string{"account": {"account": "one"}}, "missing", 0, 0},
		{"unsupported registrar role", `D("example.com", REGISTRAR("account"));`,
			map[string]map[string]string{"account": {"TYPE": "BIND"}}, "no such registrar type", 0, 0},
		{"mixed explicit type without credentials", `NewRegistrar("account", "TEST_PROVIDER_SYNTAX"); D("example.com", REGISTRAR("account"));`,
			nil, "", 1, 0},
		{"mixed explicit type without TYPE", `NewDnsProvider("account", "TEST_PROVIDER_SYNTAX"); D("example.com", REGISTRAR("none"), SERVICE("account"));`,
			map[string]map[string]string{"none": {"TYPE": "NONE"}, "account": {"account": "one"}}, "", 0, 1},
		{"mixed explicit type mismatch", `NewRegistrar("account", "TEST_PROVIDER_SYNTAX"); D("example.com", REGISTRAR("account"));`,
			map[string]map[string]string{"account": {"TYPE": "NONE"}}, "Mismatch", 0, 0},
		{"alias credentials for both roles", `D("example.com", REGISTRAR("account"), SERVICE("account"));`,
			map[string]map[string]string{"account": {"TYPE": "TEST_PROVIDER_SYNTAX_ALIAS"}}, "", 1, 1},
		{"explicit alias with canonical credentials", `NewRegistrar("account", "TEST_PROVIDER_SYNTAX_ALIAS"); NewDnsProvider("account", "TEST_PROVIDER_SYNTAX_ALIAS"); D("example.com", "account", DnsProvider("account"));`,
			map[string]map[string]string{"account": {"TYPE": providerSyntaxTestType}}, "", 1, 1},
		{"explicit canonical with alias credentials", `NewRegistrar("account", "TEST_PROVIDER_SYNTAX"); NewDnsProvider("account", "TEST_PROVIDER_SYNTAX"); D("example.com", "account", DnsProvider("account"));`,
			map[string]map[string]string{"account": {"TYPE": "TEST_PROVIDER_SYNTAX_ALIAS"}}, "", 1, 1},
	}
	for _, initializer := range syntaxInitializers {
		t.Run(initializer.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					registrars, dns := registerSyntaxTestProvider(t)
					cfg := loadSyntaxConfig(t, tt.script)
					err := initializer.init(cfg, tt.creds)
					if tt.wantError == "" {
						require.NoError(t, err)
						for _, d := range cfg.Domains {
							require.NotEqual(t, "TEST_PROVIDER_SYNTAX_ALIAS", d.RegistrarInstance.ProviderType)
							for _, p := range d.DNSProviderInstances {
								require.Equal(t, providerSyntaxTestType, p.ProviderType)
							}
						}
					} else {
						require.ErrorContains(t, err, tt.wantError)
					}
					require.Len(t, *registrars, tt.wantReg)
					require.Len(t, *dns, tt.wantDNS)
				})
			}
		})
	}
}

func TestProviderConversionGuide(t *testing.T) {
	guide, err := os.ReadFile("../documentation/getting-started/converting-dnsconfig.md")
	require.NoError(t, err)
	examples := regexp.MustCompile("(?s)```javascript\\n(.*?)```").FindAllStringSubmatch(string(guide), -1)
	require.Len(t, examples, 4, "two complete before/after pairs")
	credsPath := filepath.Join(t.TempDir(), "creds.json")
	require.NoError(t, os.WriteFile(credsPath, []byte(`{"gandi_main":{"TYPE":"GANDI_V5"}}`), 0o600))
	creds, err := credsfile.LoadProviderConfigs(credsPath)
	require.NoError(t, err)
	for i := 0; i < len(examples); i += 2 {
		before := loadSyntaxConfig(t, examples[i][1])
		after := loadSyntaxConfig(t, examples[i+1][1])
		_, err := populateProviderTypes(before, creds)
		require.NoError(t, err)
		_, err = populateProviderTypes(after, creds)
		require.NoError(t, err)
		// Moving metadata into SERVICE changes its IR location, not its effective
		// value. Compare configs after resolving legacy metadata into domains.
		for _, cfg := range []*models.DNSConfig{before, after} {
			for _, domain := range cfg.Domains {
				domain.DNSProviderMetadata = map[string]json.RawMessage{}
				for _, instance := range domain.DNSProviderInstances {
					if len(instance.Metadata) != 0 {
						domain.DNSProviderMetadata[instance.Name] = instance.Metadata
					}
				}
			}
			for _, provider := range cfg.DNSProviders {
				provider.Metadata = nil
			}
		}
		beforeJSON, err := json.Marshal(before)
		require.NoError(t, err)
		afterJSON, err := json.Marshal(after)
		require.NoError(t, err)
		require.JSONEq(t, string(beforeJSON), string(afterJSON))
	}
}

func TestProviderSyntaxIRLoading(t *testing.T) {
	// Role declarations and domain metadata must survive direct IR loading
	// without another JavaScript finalization pass.
	cfg := loadSyntaxConfig(t, `D("example.com", REGISTRAR("both"), SERVICE("both", 0, {nested:[1,null]}));`)
	want, err := json.Marshal(cfg)
	require.NoError(t, err)
	irPath := filepath.Join(t.TempDir(), "dnsconfig.json")
	require.NoError(t, os.WriteFile(irPath, want, 0o600))
	roundTrip, err := GetDNSConfig(GetDNSConfigArgs{JSONFile: irPath})
	require.NoError(t, err)
	got, err := json.Marshal(roundTrip)
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(got))
	require.Equal(t, "both", roundTrip.Domains[0].RegistrarInstance.Name)
	require.Equal(t, "both", roundTrip.Domains[0].DNSProviderInstances[0].Name)
}

func TestServiceSyntaxDomainMetadata(t *testing.T) {
	for _, initializer := range syntaxInitializers {
		t.Run(initializer.name, func(t *testing.T) {
			registrars, dns := registerSyntaxTestProvider(t)
			cfg := loadSyntaxConfig(t, `
				DEFAULTS(REGISTRAR("account"));
				D("example.com", SERVICE("account", ALL_NS, {setting:"one"}));
				D("example.net", SERVICE("account", 0, {setting:"two"}));
				D("example.org");
				setTimeout(function() {
					D_EXTEND("example.org", SERVICE("account", 2, {setting:"one"}));
				}, 1);`)
			creds := map[string]map[string]string{"account": {"TYPE": providerSyntaxTestType}}
			require.NoError(t, initializer.init(cfg, creds))
			require.Len(t, *registrars, 1)
			require.Empty(t, (*registrars)[0].metadata)
			require.Len(t, *dns, 2)
			require.JSONEq(t, `{"setting":"one"}`, string((*dns)[0].metadata))
			require.JSONEq(t, `{"setting":"two"}`, string((*dns)[1].metadata))
			require.Same(t, cfg.Domains[0].DNSProviderInstances[0].Driver, cfg.Domains[2].DNSProviderInstances[0].Driver)
			require.NotSame(t, cfg.Domains[0].DNSProviderInstances[0].Driver, cfg.Domains[1].DNSProviderInstances[0].Driver)
		})
	}
}

func TestServiceSyntaxDefaultCredentials(t *testing.T) {
	for _, contents := range []string{"missing", "", " \n\t", "{}"} {
		t.Run(fmt.Sprintf("%q", contents), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "creds.json")
			if contents != "missing" {
				require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
			}
			creds, err := credsfile.LoadProviderConfigs(path)
			require.NoError(t, err)
			for _, initializer := range syntaxInitializers {
				t.Run(initializer.name, func(t *testing.T) {
					cfg := loadSyntaxConfig(t, `D("example.com", REGISTRAR("none"), SERVICE("bind", 0));`)
					require.NoError(t, initializer.init(cfg, creds))
					require.Equal(t, "NONE", cfg.Domains[0].RegistrarInstance.ProviderType)
					require.Equal(t, "BIND", cfg.Domains[0].DNSProviderInstances[0].ProviderType)
				})
			}
		})
	}
}
