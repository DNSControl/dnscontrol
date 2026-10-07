package js

import (
	"testing"

	testifyrequire "github.com/stretchr/testify/require"
)

func TestServiceMetadata(t *testing.T) {
	for _, value := range []string{`{"nested":[1,true,null]}`, `[1,"two"]`, `"text"`, `42`, `false`, `null`} {
		t.Run(value, func(t *testing.T) {
			cfg := parseProviderConfig(t, `D("example.com", REGISTRAR("none"), SERVICE("__proto__", ALL_NS, `+value+`));`)
			testifyrequire.JSONEq(t, value, string(cfg.Domains[0].DNSProviderMetadata["__proto__"]))
			testifyrequire.Empty(t, cfg.Registrars[0].Metadata)
			testifyrequire.Empty(t, cfg.DNSProviders[0].Metadata)
			testifyrequire.Equal(t, -1, cfg.Domains[0].DNSProviderNames["__proto__"])
		})
	}
	cfg := parseProviderConfig(t, `
		var meta = {setting:"original"};
		DEFAULTS(REGISTRAR("none"), SERVICE("dns", ALL_NS, meta));
		meta.setting = "changed";
		D("example.com");
		D("example.net", SERVICE("other", 0, {setting:"other"}));
		D_EXTEND("sub.example.com", SERVICE("extra", ALL_NS, {setting:"extended"}));`)
	testifyrequire.JSONEq(t, `{"setting":"original"}`, string(cfg.Domains[0].DNSProviderMetadata["dns"]))
	testifyrequire.JSONEq(t, `{"setting":"original"}`, string(cfg.Domains[1].DNSProviderMetadata["dns"]))
	testifyrequire.JSONEq(t, `{"setting":"other"}`, string(cfg.Domains[1].DNSProviderMetadata["other"]))
	testifyrequire.JSONEq(t, `{"setting":"extended"}`, string(cfg.Domains[0].DNSProviderMetadata["extra"]))

	cfg = parseProviderConfig(t, `
		D("example.com", REGISTRAR("none"), SERVICE("dns", ALL_NS, {setting:"one"}));
		D("example.net", REGISTRAR("none"), SERVICE("dns", 0, {setting:"two"}));`)
	testifyrequire.JSONEq(t, `{"setting":"one"}`, string(cfg.Domains[0].DNSProviderMetadata["dns"]))
	testifyrequire.JSONEq(t, `{"setting":"two"}`, string(cfg.Domains[1].DNSProviderMetadata["dns"]))

	cfg = parseProviderConfig(t, `
		D("example.com", REGISTRAR("none"), SERVICE("dns", ALL_NS,
			{ip_conversions:[{low:IP("192.0.2.0"),high:IP("192.0.2.255"),newBase:IP("198.51.100.0")}]}));`)
	legacy := parseProviderConfig(t, `NewDnsProvider("dns",
		{ip_conversions:[{low:IP("192.0.2.0"),high:IP("192.0.2.255"),newBase:IP("198.51.100.0")}]});`)
	testifyrequire.JSONEq(t, string(legacy.DNSProviders[0].Metadata), string(cfg.Domains[0].DNSProviderMetadata["dns"]))
}

func TestUnusedProviderDefaults(t *testing.T) {
	cfg := parseProviderConfig(t, `REGISTRAR("unused"); SERVICE("unused", ALL_NS, {a:1});
		DEFAULTS(REGISTRAR("unused"), SERVICE("unused"));`)
	testifyrequire.Empty(t, cfg.Registrars)
	testifyrequire.Empty(t, cfg.DNSProviders)
}
