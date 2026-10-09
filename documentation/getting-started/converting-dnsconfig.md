# The new `D()` syntax

This experimental syntax selects providers directly using their entry names in
creds.json. The legacy `NewRegistrar`, `NewDnsProvider`, and `DnsProvider` helpers
remain supported.

Before:

```javascript
var REG_GANDI = NewRegistrar("gandi_main");
var DSP_GANDI = NewDnsProvider("gandi_main");

D("example.com", REG_GANDI, DnsProvider(DSP_GANDI),
    A("@", "192.0.2.1")
);
```

After:

```javascript
D("example.com",
    REGISTRAR("gandi_main"),
    SERVICE("gandi_main"),
    A("@", "192.0.2.1")
);
```

`REGISTRAR()` must immediately follow the domain name. It may also appear in
`DEFAULTS()`; an explicit registrar overrides that default. Its only optional
argument is the credential entry name, with no metadata. With no argument,
`REGISTRAR()` selects `"none"`.

`SERVICE(name, maxNS, configMetadata)` takes an optional nameserver limit and
optional metadata. Omit maxNS to use all nameservers, use `0` to use none, or
use `ALL_NS` (equal to `-1`) when supplying metadata without a limit.

## Moving provider metadata

Before:

```javascript
var DSP_GANDI = NewDnsProvider("gandi_main", {setting: "value"});
var REG_GANDI = NewRegistrar("gandi_main");

D("example.com", REG_GANDI, DnsProvider(DSP_GANDI),
    A("@", "192.0.2.1")
);
```

After:

```javascript
D("example.com",
    REGISTRAR("gandi_main"),
    SERVICE("gandi_main", ALL_NS, {setting: "value"}),
    A("@", "192.0.2.1")
);
```

Metadata may be any JSON value and must follow maxNS. Declare it only once per
domain and credential entry. Remove the metadata-bearing `NewDnsProvider()`
declaration when moving metadata to `SERVICE()`: keeping both is an error even
if their values match. Different domains may use different metadata for the
same credential entry.

## Substitutions

| Before | After |
| --- | --- |
| `D(name, REG, ...)` | `D(name, REGISTRAR("credEntry"), ...)` |
| `DnsProvider(DSP)` | `SERVICE("credEntry")` |
| `DnsProvider(DSP, maxNS)` | `SERVICE("credEntry", maxNS)` |
| `NewDnsProvider("credEntry", metadata)` | Move metadata to `SERVICE("credEntry", ALL_NS, metadata)` for each domain. |
| `NewRegistrar(...)` / `NewDnsProvider(...)` variables | Remove declarations once all their uses are converted. |
| Experimental `PROVIDER("credEntry")` and `DNS_SERVICE(...)` from v5.3.0 | Remove the declaration; use the entry name directly with `REGISTRAR` / `SERVICE`. Move `PROVIDER` metadata to each `SERVICE`. |

Leave creds.json unchanged, with provider `TYPE` values there. Empty or missing
credential files still supply the default `none` and `bind` entries. These
modifiers also work in `DEFAULTS()` and `D_EXTEND()`; an explicit `REGISTRAR` in
`D_EXTEND()` must likewise follow the domain name.

## Feedback needed

The v5.3.0 syntax was experimental; `PROVIDER` and `DNS_SERVICE` have been
replaced. Please test this revision and [share feedback](https://github.com/orgs/DNSControl/discussions/4949).

Compare `dnscontrol preview` before and after conversion. The output should be the same. [Report problems or confusing conversions](https://github.com/orgs/DNSControl/discussions/4949) with your DNSControl version, provider type, and a minimal example without credentials.
