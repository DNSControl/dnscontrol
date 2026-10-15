# The new `D()` syntax

This new syntax selects providers directly using their entry names in
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

`REGISTRAR()`:
* Must immediately follow the domain name. 
* Its only argument is the credential entry name (no metadata).
* `REGISTRAR()` is an alias for `REGISTRAR("none")`.
* `REGISTRAR()` may appear in `DEFAULTS()`, but setting a `REGISTRAR()` in a `D()` will override it.

`SERVICE(name, maxNS, configMetadata)`:
* Takes an optional nameserver limit and optional metadata.
* Omit maxNS to use all nameservers, use `0` to use none, or use `ALL_NS` when supplying metadata without a limit.

## Converting provider metadata

Metadata that was previously supplied via `NewDnsProvider()` now is included in `SERVICE()` instead.

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

Metadata may be any JSON value and must follow maxNS. 

Including metadata in both `NewDnsProvider()` and `SERVICE()` is an error.

## Substitutions

| Before | After |
| --- | --- |
| `D(name, REG, ...)` | `D(name, REGISTRAR("credEntry"), ...)` |
| `DnsProvider(DSP)` | `SERVICE("credEntry")` |
| `DnsProvider(DSP, maxNS)` | `SERVICE("credEntry", maxNS)` |
| `NewDnsProvider("credEntry", metadata)` | Move metadata to `SERVICE("credEntry", ALL_NS, metadata)` for each domain. |
| `NewRegistrar(...)` / `NewDnsProvider(...)` variables | Remove declarations once all their uses are converted. |
| Experimental `PROVIDER("credEntry")` and `DNS_SERVICE(...)` from v5.3.0 | Remove the declaration; use the entry name directly with `REGISTRAR` / `SERVICE`. Move `PROVIDER` metadata to each `SERVICE`. |

Leave creds.json unchanged.

`REGISTRAR()` and `SERVICE()` also work in `DEFAULTS()`. Only `SERVICE()` works
in `D_EXTEND()`.

## Advice

Compare `dnscontrol preview` before and after conversion. The output should be the same. [Report problems or confusing conversions](https://github.com/orgs/DNSControl/discussions/4949) with your DNSControl version, provider type, and a minimal example without credentials.
