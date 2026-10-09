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
			modern: `var r = "reg"; var a = "a"; var b = "b"; var c = "c";
				D("example.com", REGISTRAR(r), SERVICE(a), SERVICE(b, 0), SERVICE(c, 2), A("@", "192.0.2.1"));`,
		},
		{
			name: "one entry both roles",
			legacy: `var r = NewRegistrar("both"); var d = NewDnsProvider("both");
				D("example.com", r, DnsProvider(d), A("@", "192.0.2.1"));`,
			modern: `var p = "both";
				D("example.com", REGISTRAR(p), SERVICE(p), A("@", "192.0.2.1"));`,
		},
		{
			name:   "omitted registrar name selects none",
			legacy: `NewRegistrar("none"); NewDnsProvider("bind"); D("example.com", "none", DnsProvider("bind", 0));`,
			modern: `D("example.com", REGISTRAR(), SERVICE("bind", 0));`,
		},
		{
			name:   "omitted registrar name in defaults",
			legacy: `NewRegistrar("none"); NewDnsProvider("bind"); D("example.com", "none", DnsProvider("bind", 0));`,
			modern: `DEFAULTS(REGISTRAR(), SERVICE("bind", 0)); D("example.com");`,
		},
		{
			name:   "new modifiers with legacy declarations",
			legacy: `NewRegistrar("r", "NONE"); NewDnsProvider("d", "BIND"); D("example.com", "r", DnsProvider("d"));`,
			modern: `NewRegistrar("r", "NONE"); NewDnsProvider("d", "BIND"); D("example.com", REGISTRAR("r"), SERVICE("d"));`,
		},
		{
			name: "defaults arrays and metadata",
			legacy: `NewRegistrar("reg"); NewDnsProvider("dns");
				DEFAULTS([DnsProvider("dns"), DefaultTTL(600)], {flag: "value"});
				D("example.com", "reg", [A("@", "192.0.2.1")], {other: "value"});`,
			modern: `DEFAULTS([REGISTRAR("reg"), SERVICE("dns"), DefaultTTL(600)], {flag: "value"});
				D("example.com", [A("@", "192.0.2.1")], {other: "value"});`,
		},
		{
			name:   "explicit registrar overrides default",
			legacy: `NewRegistrar("chosen"); D("example.com", "chosen"); D("example.net", "chosen");`,
			modern: `DEFAULTS(REGISTRAR("unused"));
				D("example.com", REGISTRAR("chosen")); D("example.net", "chosen");`,
		},
		{
			name:   "extension overrides default",
			legacy: `NewRegistrar("chosen"); NewDnsProvider("dns"); D("example.com", "chosen", DnsProvider("dns", 0));`,
			modern: `DEFAULTS(REGISTRAR("unused"));
				D("example.com"); D_EXTEND("example.com", REGISTRAR("chosen"), SERVICE("dns", 0));`,
		},
		{
			name:   "registrar supplied after domain declaration",
			legacy: `NewRegistrar("reg"); D("example.com", "reg", A("@", "192.0.2.1"));`,
			modern: `D("example.com", A("@", "192.0.2.1"));
				D_EXTEND("example.com", REGISTRAR("reg"));`,
		},
		{
			name:   "asynchronous finalization",
			legacy: `NewRegistrar("reg"); NewDnsProvider("dns"); D("example.com", "reg", DnsProvider("dns"));`,
			modern: `D("example.com");
				setTimeout(function () { D_EXTEND("example.com", REGISTRAR("reg"), SERVICE("dns")); }, 1);`,
		},
		{
			name:   "promise finalization",
			legacy: `NewRegistrar("reg"); D("example.com", "reg");`,
			modern: `Promise.resolve().then(function () { D("example.com", REGISTRAR("reg")); }).catch(PANIC);`,
		},
		{
			name: "mixed declarations preserve explicit types and role metadata",
			legacy: `NewRegistrar("both", "TYPE", {role: "registrar"}); NewDnsProvider("both", "TYPE", {role: "dns"});
				D("example.com", "both", DnsProvider("both"));`,
			modern: `NewRegistrar("both", "TYPE", {role: "registrar"});
				NewDnsProvider("both", "TYPE", {role: "dns"}); D("example.com", REGISTRAR("both"), SERVICE("both"));`,
		},
		{
			name:   "object property names are valid credential names",
			legacy: `NewRegistrar("constructor"); NewDnsProvider("toString"); D("example.com", "constructor", DnsProvider("toString"));`,
			modern: `D("example.com", REGISTRAR("constructor"), SERVICE("toString"));`,
		},
		{
			name:   "legacy globals may shadow new functions",
			legacy: `NewRegistrar("reg"); NewDnsProvider("dns"); DOMAIN_ELSEWHERE_AUTO("example.com", "reg", "dns");`,
			modern: `var REGISTRAR = NewRegistrar("reg"); var SERVICE = NewDnsProvider("dns");
				DOMAIN_ELSEWHERE_AUTO("example.com", REGISTRAR, SERVICE);`,
		},
		{
			name:   "elsewhere auto accepts service modifier",
			legacy: `NewRegistrar("reg"); NewDnsProvider("dns"); DOMAIN_ELSEWHERE_AUTO("example.com", "reg", "dns");`,
			modern: `DOMAIN_ELSEWHERE_AUTO("example.com", REGISTRAR("reg"), SERVICE("dns"));`,
		},
		{
			name:   "custom modifier can clear DNS services",
			legacy: `NewRegistrar("reg"); D("example.com", "reg", function(d) { d.dnsProviders = null; });`,
			modern: `D("example.com", REGISTRAR("reg"), SERVICE("unused"), function(d) { d.dnsProviders = null; });`,
		},
		{
			name:   "mixed declarations retain legacy duplicates",
			legacy: `NewRegistrar("reg", "FIRST"); NewRegistrar("reg", "SECOND"); D("example.com", "reg");`,
			modern: `NewRegistrar("reg", "FIRST"); NewRegistrar("reg", "SECOND"); D("example.com", REGISTRAR("reg"));`,
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
		{"no inferred registrar", `D("example.com", SERVICE("both"));`, "requires a registrar"},
		{"late registrar", `D("example.com", SERVICE("dns"), REGISTRAR("reg"));`, "must immediately follow"},
		{"registrar in array", `D("example.com", [REGISTRAR("reg")]);`, "must immediately follow"},
		{"second registrar", `D("example.com", REGISTRAR("one"), REGISTRAR("two"));`, "must immediately follow"},
		{"positional conflict", `D("example.com", "one", REGISTRAR("two"));`, "must immediately follow"},
		{"extension conflict", `D("example.com", REGISTRAR("one")); D_EXTEND("example.com", REGISTRAR("two"));`, "Conflicting registrars"},
		{"late extension registrar", `D("example.com", REGISTRAR("reg")); D_EXTEND("example.com", A("@","192.0.2.1"), REGISTRAR("reg"));`, "must immediately follow"},
		{"async conflict", `D("example.com", REGISTRAR("one")); setTimeout(function() {D_EXTEND("example.com", REGISTRAR("two"));},1);`, "Conflicting registrars"},
		{"conflicting defaults", `DEFAULTS(REGISTRAR("one"), REGISTRAR("two")); D("example.com");`, "Conflicting registrars"},
		{"empty registrar", `REGISTRAR("");`, "nonempty credential entry name"},
		{"registrar metadata", `REGISTRAR("reg", {});`, "configMetadata is not supported"},
		{"empty service", `SERVICE("");`, "nonempty credential entry name"},
		{"nonstring service", `SERVICE({});`, "nonempty credential entry name"},
		{"metadata without count", `SERVICE("dns", {});`, "configMetadata must follow maxNS"},
		{"undefined count with metadata", `SERVICE("dns", undefined, {});`, "configMetadata must follow maxNS"},
		{"fractional count", `SERVICE("dns", 1.5);`, "maxNS"},
		{"negative count", `SERVICE("dns", -2);`, "maxNS"},
		{"infinite count", `SERVICE("dns", Infinity);`, "maxNS"},
		{"NaN count", `SERVICE("dns", NaN);`, "maxNS"},
		{"string count", `SERVICE("dns", "2");`, "maxNS"},
		{"extra argument", `SERVICE("dns", 0, {}, {});`, "SERVICE accepts"},
		{"function metadata", `SERVICE("dns", 0, function(){});`, "must be a JSON value"},
		{"duplicate same metadata", `D("example.com", REGISTRAR("none"), SERVICE("dns", 0, {}), SERVICE("dns", 0, {}));`, "SERVICE() and SERVICE()"},
		{"duplicate different metadata", `D("example.com", REGISTRAR("none"), SERVICE("dns", 0, {a:1}), SERVICE("dns", 0, {a:2}));`, "duplicate configMetadata error"},
		{"duplicate default metadata", `DEFAULTS(REGISTRAR("none"), SERVICE("dns", 0, {})); D("example.com", SERVICE("dns", 0, {}));`, "duplicate configMetadata error"},
		{"duplicate extension metadata", `D("example.com", REGISTRAR("none"), SERVICE("dns", 0, null)); D_EXTEND("example.com", SERVICE("dns", 0, null));`, "duplicate configMetadata error"},
		{"unused legacy declaration duplicate", `var unused = NewDnsProvider("dns", {}); D("example.com", REGISTRAR("none"), SERVICE("dns", 0, {}));`, "NewDnsProvider() and SERVICE()"},
		{"null legacy metadata duplicate", `NewDnsProvider("dns", "-", null); D("example.com", REGISTRAR("none"), SERVICE("dns", 0, null));`, "NewDnsProvider() and SERVICE()"},
		{"late legacy declaration duplicate", `D("example.com", REGISTRAR("none"), SERVICE("dns", 0, {})); NewDnsProvider("dns", {});`, "NewDnsProvider() and SERVICE()"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExecuteJavascriptString([]byte(tt.script), false, nil)
			testifyrequire.ErrorContains(t, err, tt.message)
		})
	}
}

func TestProviderSyntaxPrototypeName(t *testing.T) {
	cfg := parseProviderConfig(t, `D("example.com", REGISTRAR("__proto__"), SERVICE("__proto__", 0));`)
	testifyrequire.Len(t, cfg.Registrars, 1)
	testifyrequire.Len(t, cfg.DNSProviders, 1)
	testifyrequire.Equal(t, map[string]int{"__proto__": 0}, cfg.Domains[0].DNSProviderNames)
}
