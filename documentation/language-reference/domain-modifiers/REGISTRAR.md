---
name: REGISTRAR
parameters:
  - credEntry
parameter_types:
  credEntry: string?
---

`REGISTRAR(credEntry)` selects the registrar using an entry name from creds.json.

No `NewRegistrar()` declaration is needed.

`REGISTRAR()` is an alias for `REGISTRAR("none")`, which disables registrar management.

In `D()`, `REGISTRAR()` must immediately follow the domain name.

```javascript
D("example.com", REGISTRAR("gandi_main"),
    SERVICE("gandi_main"),
    A("@", "192.0.2.1"),
);

D("example.net", REGISTRAR(),
    SERVICE("gandi_main"),
    A("@", "192.0.2.1"),
);
```

`REGISTRAR()` may appear in `DEFAULTS()`, but setting a `REGISTRAR()` in `D()` or
`D_EXTEND()` will override it for that domain.

```javascript
DEFAULTS(
    REGISTRAR(),
    SERVICE("bind", 0),
);

D("example.com", A("@", "192.0.2.1"));
D("example.net", REGISTRAR("gandi_main"), A("@", "192.0.2.2"));
```

`REGISTRAR()` may appear anywhere among the modifiers in
[`D_EXTEND()`](../top-level-functions/D_EXTEND.md), including inside nested arrays.
It replaces the registrar previously selected by defaults, `D()` (including the
legacy positional form), or an earlier extension. The last registrar assignment
executed wins. Different registrar selections within a single `D_EXTEND()` call
are an error; repeating the same selection is allowed.

```javascript
D("example.org", REGISTRAR("old_registrar"), SERVICE("dns_main"));
D_EXTEND("example.org",
    A("@", "192.0.2.3"),
    REGISTRAR("new_registrar"),
);

// A later extension can disable registrar management.
D_EXTEND("example.org", REGISTRAR());
```

A domain may also receive its first registrar selection through `D_EXTEND()`.
Registrar selection is finalized after the script and its asynchronous callbacks
finish. A domain still missing a required registrar at that point is an error.
Registrar changes leave DNS services and their metadata unchanged.
`REGISTRAR()` does not accept configuration metadata.

The legacy [NewRegistrar](../top-level-functions/NewRegistrar.md) helper remains
supported. See the [conversion guide](../../getting-started/converting-dnsconfig.md).
