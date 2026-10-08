---
name: REGISTRAR
parameters:
  - credEntry
parameter_types:
  credEntry: string
---

`REGISTRAR(credEntry)` selects the registrar using an entry name from creds.json.
The entry's `TYPE` determines the provider. No `REGISTRAR()` declaration is
needed. Use `REGISTRAR("none")` when DNSControl should not manage registration.

An explicit `REGISTRAR()` must immediately follow the domain name in `D()` or
`D_EXTEND()`. It accepts no configuration metadata.

```javascript
D("example.com", REGISTRAR("gandi_main"), SERVICE("gandi_main"),
    A("@", "192.0.2.1"),
);
```

It may also appear in `DEFAULTS()`. A domain's explicit registrar overrides the
default; conflicting explicit selections are errors. A domain needs an explicit
registrar or a default, even when the DNS service uses the same credential entry.

```javascript
DEFAULTS(REGISTRAR("none"), SERVICE("bind", 0));
D("example.com", A("@", "192.0.2.1"));
D("example.net", REGISTRAR("gandi_main"), A("@", "192.0.2.2"));
```

The legacy [NewRegistrar](../top-level-functions/NewRegistrar.md) helper remains
supported. See the [conversion guide](../../getting-started/converting-dnsconfig.md).
