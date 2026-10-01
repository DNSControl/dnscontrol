package js

import (
	"encoding/json"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	testifyrequire "github.com/stretchr/testify/require"
)

func parseProviderConfig(t *testing.T, script string) *models.DNSConfig {
	t.Helper()
	cfg, err := ExecuteJavascriptString([]byte(script), false, nil)
	testifyrequire.NoError(t, err)
	// Equivalent syntax has different source positions; compare the DNS content.
	for _, domain := range cfg.Domains {
		for _, record := range domain.Records {
			record.FilePos = ""
		}
	}
	return cfg
}

func TestProviderSyntaxCompatibility(t *testing.T) {
	tests := []struct {
		name   string
		legacy string
		modern string
	}{
		{
			name: "separate roles and nameserver counts",
			legacy: `var r = NewRegistrar("reg"); var a = NewDnsProvider("a"); var b = NewDnsProvider("b"); var c = NewDnsProvider("c");
				D("example.com", r, DnsProvider(a), DnsProvider(b, 0), DnsProvider(c, 2), A("@", "192.0.2.1"));`,
			modern: `var r = PROVIDER("reg"); var a = PROVIDER("a"); var b = PROVIDER("b"); var c = PROVIDER("c");
				D("example.com", REGISTRAR(r), DNS_SERVICE(a), DNS_SERVICE(b, 0), DNS_SERVICE(c, 2), A("@", "192.0.2.1"));`,
		},
		{
			name: "one entry both roles",
			legacy: `var r = NewRegistrar("both"); var d = NewDnsProvider("both");
				D("example.com", r, DnsProvider(d), A("@", "192.0.2.1"));`,
			modern: `var p = PROVIDER("both");
				if (typeof p !== "string" || p !== "both") { throw "PROVIDER must return the credential entry name"; }
				D("example.com", REGISTRAR(p), DNS_SERVICE(p), A("@", "192.0.2.1"));`,
		},
		{
			name:   "positional registrar and legacy modifier",
			legacy: `NewRegistrar("both"); NewDnsProvider("both"); D("example.com", "both", DnsProvider("both"));`,
			modern: `PROVIDER("both"); D("example.com", "both", DnsProvider("both"));`,
		},
		{
			name:   "new modifiers with legacy declarations",
			legacy: `NewRegistrar("r", "NONE"); NewDnsProvider("d", "BIND"); D("example.com", "r", DnsProvider("d"));`,
			modern: `NewRegistrar("r", "NONE"); NewDnsProvider("d", "BIND"); D("example.com", REGISTRAR("r"), DNS_SERVICE("d"));`,
		},
		{
			name: "defaults arrays and metadata",
			legacy: `NewRegistrar("reg"); NewDnsProvider("dns");
				DEFAULTS([DnsProvider("dns"), DefaultTTL(600)], {flag: "value"});
				D("example.com", "reg", [A("@", "192.0.2.1")], {other: "value"});`,
			modern: `PROVIDER("reg"); PROVIDER("dns");
				DEFAULTS([REGISTRAR("reg"), DNS_SERVICE("dns"), DefaultTTL(600)], {flag: "value"});
				D("example.com", [A("@", "192.0.2.1")], {other: "value"});`,
		},
		{
			name:   "explicit registrar overrides default",
			legacy: `NewRegistrar("chosen"); D("example.com", "chosen"); D("example.net", "chosen");`,
			modern: `PROVIDER("unused"); PROVIDER("chosen"); DEFAULTS(REGISTRAR("unused"));
				D("example.com", REGISTRAR("chosen")); D("example.net", "chosen");`,
		},
		{
			name:   "extension overrides default",
			legacy: `NewRegistrar("chosen"); NewDnsProvider("dns"); D("example.com", "chosen", DnsProvider("dns", 0));`,
			modern: `PROVIDER("unused"); PROVIDER("chosen"); PROVIDER("dns"); DEFAULTS(REGISTRAR("unused"));
				D("example.com"); D_EXTEND("example.com", REGISTRAR("chosen"), DNS_SERVICE("dns", 0));`,
		},
		{
			name:   "registrar supplied after domain declaration",
			legacy: `NewRegistrar("reg"); D("example.com", "reg", A("@", "192.0.2.1"));`,
			modern: `D("example.com", A("@", "192.0.2.1"));
				D_EXTEND("example.com", REGISTRAR(PROVIDER("reg")));`,
		},
		{
			name:   "asynchronous finalization",
			legacy: `NewRegistrar("reg"); NewDnsProvider("dns"); D("example.com", "reg", DnsProvider("dns"));`,
			modern: `PROVIDER("reg"); D("example.com");
				setTimeout(function () { PROVIDER("dns"); D_EXTEND("example.com", REGISTRAR("reg"), DNS_SERVICE("dns")); }, 1);`,
		},
		{
			name:   "promise finalization",
			legacy: `NewRegistrar("reg"); D("example.com", "reg");`,
			modern: `Promise.resolve().then(function () { D("example.com", REGISTRAR(PROVIDER("reg"))); }).catch(PANIC);`,
		},
		{
			name: "domain helpers and includes",
			legacy: `NewRegistrar("reg"); NewDnsProvider("dns");
				DOMAIN_ELSEWHERE("example.com", "reg", ["ns1.example.org"]);
				DOMAIN_ELSEWHERE_AUTO("example.net", "reg", "dns");
				D("example.org", "reg", TXT("@", "hello"));
				D_EXTEND("example.net", INCLUDE("example.org"));`,
			modern: `PROVIDER("reg"); PROVIDER("dns");
				DOMAIN_ELSEWHERE("example.com", "reg", ["ns1.example.org"]);
				DOMAIN_ELSEWHERE_AUTO("example.net", "reg", "dns");
				D("example.org", REGISTRAR("reg"), TXT("@", "hello"));
				D_EXTEND("example.net", INCLUDE("example.org"));`,
		},
		{
			name: "role metadata and IP conversions",
			legacy: `var metadata = {ip_conversions: [{low: IP("192.0.2.0"), high: IP("192.0.2.255"), newBase: IP("198.51.100.0")}]};
				NewRegistrar("both", JSON.parse(JSON.stringify(metadata))); NewDnsProvider("both", metadata);
				D("example.com", "both", DnsProvider("both"));`,
			modern: `PROVIDER("both", {ip_conversions: [{low: IP("192.0.2.0"), high: IP("192.0.2.255"), newBase: IP("198.51.100.0")}]});
				D("example.com", REGISTRAR("both"), DNS_SERVICE("both"));`,
		},
		{
			name: "shared legacy metadata already has formatted conversions",
			legacy: `NewRegistrar("reg"); NewDnsProvider("dns", {ip_conversions: [{low: IP("192.0.2.0"), high: IP("192.0.2.255"), newBase: IP("198.51.100.0")}]});
				D("example.com", "reg", DnsProvider("dns"));`,
			modern: `PROVIDER("reg"); var metadata = {ip_conversions: [{low: IP("192.0.2.0"), high: IP("192.0.2.255"), newBase: IP("198.51.100.0")}]};
				NewDnsProvider("dns", metadata); PROVIDER("dns", metadata); D("example.com", REGISTRAR("reg"), DNS_SERVICE("dns"));`,
		},
		{
			name: "mixed declarations preserve explicit types and role metadata",
			legacy: `NewRegistrar("both", "TYPE", {role: "registrar"}); NewDnsProvider("both", "TYPE", {role: "dns"});
				D("example.com", "both", DnsProvider("both"));`,
			modern: `PROVIDER("both"); NewRegistrar("both", "TYPE", {role: "registrar"});
				NewDnsProvider("both", "TYPE", {role: "dns"}); D("example.com", REGISTRAR("both"), DNS_SERVICE("both"));`,
		},
		{
			name: "matching metadata reuses legacy declaration",
			legacy: `NewRegistrar("both", "TYPE", {a: "1", b: "2"}); NewDnsProvider("both", "TYPE", {a: "1", b: "2"});
				D("example.com", "both", DnsProvider("both"));`,
			modern: `NewRegistrar("both", "TYPE", {a: "1", b: "2"}); NewDnsProvider("both", "TYPE", {a: "1", b: "2"});
				PROVIDER("both", {b: "2", a: "1"}); D("example.com", REGISTRAR("both"), DNS_SERVICE("both"));`,
		},
		{
			name:   "identical neutral declarations are idempotent",
			legacy: `NewRegistrar("reg"); D("example.com", "reg");`,
			modern: `PROVIDER("reg"); PROVIDER("reg"); D("example.com", REGISTRAR("reg"), REGISTRAR("reg"));`,
		},
		{
			name:   "unused declarations and defaults are not materialized",
			legacy: ``,
			modern: `PROVIDER("unused", {arbitrary: "data"}); DEFAULTS(REGISTRAR("unused"), DNS_SERVICE("unused"));`,
		},
		{
			name:   "deterministic declaration order",
			legacy: `NewRegistrar("reg"); NewDnsProvider("a"); NewDnsProvider("b"); D("example.com", "reg", DnsProvider("b"), DnsProvider("a"));`,
			modern: `PROVIDER("reg"); PROVIDER("a"); PROVIDER("b"); D("example.com", REGISTRAR("reg"), DNS_SERVICE("b"), DNS_SERVICE("a"));`,
		},
		{
			name:   "object property names are valid credential names",
			legacy: `NewRegistrar("constructor"); NewDnsProvider("toString"); D("example.com", "constructor", DnsProvider("toString"));`,
			modern: `PROVIDER("constructor"); PROVIDER("toString"); D("example.com", REGISTRAR("constructor"), DNS_SERVICE("toString"));`,
		},
		{
			name:   "legacy globals may shadow new functions",
			legacy: `NewRegistrar("reg"); NewDnsProvider("dns"); DOMAIN_ELSEWHERE_AUTO("example.com", "reg", "dns");`,
			modern: `var REGISTRAR = NewRegistrar("reg"); var PROVIDER = NewDnsProvider("dns"); var DNS_SERVICE = "anything";
				DOMAIN_ELSEWHERE_AUTO("example.com", REGISTRAR, PROVIDER);`,
		},
		{
			name:   "custom modifier can clear DNS services",
			legacy: `NewRegistrar("reg"); D("example.com", "reg", function(d) { d.dnsProviders = null; });`,
			modern: `PROVIDER("reg"); PROVIDER("unused"); D("example.com", REGISTRAR("reg"), DNS_SERVICE("unused"), function(d) { d.dnsProviders = null; });`,
		},
		{
			name:   "mixed declarations retain legacy duplicates",
			legacy: `NewRegistrar("reg", "FIRST"); NewRegistrar("reg", "SECOND"); D("example.com", "reg");`,
			modern: `NewRegistrar("reg", "FIRST"); NewRegistrar("reg", "SECOND"); PROVIDER("reg"); D("example.com", REGISTRAR("reg"));`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			legacy, err := json.Marshal(parseProviderConfig(t, tt.legacy))
			testifyrequire.NoError(t, err)
			modern, err := json.Marshal(parseProviderConfig(t, tt.modern))
			testifyrequire.NoError(t, err)
			testifyrequire.JSONEq(t, string(legacy), string(modern))
		})
	}
}

func TestProviderSyntaxErrors(t *testing.T) {
	tests := []struct {
		name, script, message string
	}{
		{"missing registrar", `D("example.com", A("@", "192.0.2.1"));`, "requires a registrar"},
		{"no inferred registrar", `D("example.com", DNS_SERVICE(PROVIDER("both")));`, "requires a registrar"},
		{"conflicting explicit registrars", `D("example.com", REGISTRAR("one"), REGISTRAR("two"));`, "Conflicting registrars"},
		{"positional conflict", `D("example.com", "one", REGISTRAR("two"));`, "Conflicting registrars"},
		{"extension conflict", `D("example.com", REGISTRAR("one")); D_EXTEND("example.com", REGISTRAR("two"));`, "Conflicting registrars"},
		{"asynchronous registrar conflict", `D("example.com", REGISTRAR("one")); setTimeout(function () { D_EXTEND("example.com", REGISTRAR("two")); }, 1);`, "Conflicting registrars"},
		{"conflicting defaults", `DEFAULTS(REGISTRAR("one"), REGISTRAR("two")); D("example.com");`, "Conflicting registrars"},
		{"empty name", `PROVIDER("");`, "nonempty credential entry name"},
		{"nonstring name", `PROVIDER(42);`, "nonempty credential entry name"},
		{"missing name", `PROVIDER();`, "nonempty credential entry name"},
		{"empty registrar", `REGISTRAR("");`, "nonempty credential entry name"},
		{"nonstring DNS service", `DNS_SERVICE({});`, "nonempty credential entry name"},
		{"explicit type argument", `PROVIDER("name", "TYPE");`, "Put the provider TYPE in creds.json"},
		{"three arguments", `PROVIDER("name", {}, {});`, "PROVIDER accepts"},
		{"null metadata", `PROVIDER("name", null);`, "PROVIDER accepts"},
		{"array metadata", `PROVIDER("name", []);`, "PROVIDER accepts"},
		{"duplicate declaration conflict", `PROVIDER("name", {a: 1}); PROVIDER("name", {a: 2});`, "Conflicting PROVIDER declarations"},
		{"legacy registrar metadata conflict", `NewRegistrar("name", {a: 1}); PROVIDER("name", {a: 2}); D("example.com", REGISTRAR("name"));`, "conflicts with its legacy registrar"},
		{"legacy DNS metadata conflict", `NewDnsProvider("name", {a: 1}); PROVIDER("name", {a: 2}); D("example.com", "reg", DNS_SERVICE("name"));`, "conflicts with its legacy DNS provider"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExecuteJavascriptString([]byte(tt.script), false, nil)
			testifyrequire.ErrorContains(t, err, tt.message)
		})
	}
}

func TestProviderSyntaxPrototypeName(t *testing.T) {
	cfg := parseProviderConfig(t, `PROVIDER("__proto__"); D("example.com", REGISTRAR("__proto__"), DNS_SERVICE("__proto__", 0));`)
	testifyrequire.Len(t, cfg.Registrars, 1)
	testifyrequire.Len(t, cfg.DNSProviders, 1)
	testifyrequire.Equal(t, map[string]int{"__proto__": 0}, cfg.Domains[0].DNSProviderNames)
}
