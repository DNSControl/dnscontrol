# The new `D()` syntax

## The problem

The `D()` syntax in dnsconfig.js is a bit confusing:

- Users find it confusing to define the same provider twice, once as a DSP and once as a registrar.
- ....and then use it a 3rd time with DnsProvider()
- The name DnsProvider() is confusingly similar to NewDnsProvider()
- The "New" in NewRegistrar/NewDnsProvider is odd. It makes sense from a CS perspective but not typical users.

## The solution:

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
var SVC_GANDI_MAIN = PROVIDER("gandi_main");  // `gandi_main` is the entry in creds.json

D("example.com",
    REGISTRAR(SVC_GANDI_MAIN),
    DNS_SERVICE(SVC_GANDI_MAIN),
    A("@", "192.0.2.1")
);
```

That's it!

The old sytax is still supported. If people like this the docs will be updated
to use the new syntax.

## Feedback needed

Post your comments here: https://github.com/orgs/DNSControl/discussions/4949

*We especially need feedback about the names!*  Is `DNS_SERVICE` too verbose? Is `SVC_GANDI_MAIN` too long?  What would you suggest instead?

## Substitutions

| Before | After |
| --- | --- |
| `NewRegistrar(...)` or `NewDnsProvider(...)` | `PROVIDER(...)` |
| `D(name, REG, ...)` | `D(name, REGISTRAR(REG), ...)` |
| `DnsProvider(...)` | `DNS_SERVICE(...)` |

Compare `dnscontrol preview` before and after conversion. The output should be the same. [Report problems or confusing conversions](https://github.com/orgs/DNSControl/discussions/4949) with your DNSControl version, provider type, and a minimal example without credentials.
