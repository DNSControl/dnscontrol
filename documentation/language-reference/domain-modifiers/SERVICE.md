---
name: SERVICE
parameters:
  - credEntry
  - maxNS
  - configMetadata
parameter_types:
  credEntry: string
  maxNS: number?
  configMetadata: ProviderConfigMetadata?
---

`SERVICE(credEntry, maxNS, configMetadata)` selects a DNS service using an entry
name from creds.json. Its `TYPE` determines the provider. No `SERVICE()`
declaration is needed.

- Omit maxNS, or use [ALL_NS](ALL_NS.md), to fetch and use all nameservers.
- Use `0` to manage records without fetching or delegating to those nameservers.
- Use a positive integer to fetch all nameservers but limit how many are used.

See [nameserver management](../../advanced-features/nameservers.md) for how the
DNS services' nameservers are combined and sent to the registrar.

```javascript
D("example.com", REGISTRAR("none"),
    SERVICE("cloudflare_main"),
    SERVICE("bind", 0),
    A("@", "192.0.2.1"),
);
```

Configuration metadata is optional and may be any JSON value supported by the
provider. It must follow maxNS: use `ALL_NS` when metadata is needed without a
nameserver limit. Metadata is specific to the domain and credential entry;
different domains may supply different values for the same entry.

```javascript
D("example.com", REGISTRAR("none"),
    SERVICE("bind", ALL_NS, {default_ns: ["ns1.example.net.", "ns2.example.net."]}),
    A("@", "192.0.2.1"),
);
```

Metadata may be declared only once per domain and entry. Multiple metadata-bearing
`SERVICE()` calls, including inherited defaults or extensions, are errors even
when equal. Supplying metadata in both `NewDnsProvider()` and `SERVICE()` is also
an error, including when the legacy declaration's variable is unused. Without
metadata in `SERVICE()`, a legacy declaration's metadata remains the fallback.

`SERVICE` works in `DEFAULTS()` and `D_EXTEND()` as well as `D()`. A domain without
any DNS services has no DNS records managed, which is useful for registrar-only
configuration.

The legacy [DnsProvider](DnsProvider.md) helper remains supported. See the
[conversion guide](../../getting-started/converting-dnsconfig.md).
