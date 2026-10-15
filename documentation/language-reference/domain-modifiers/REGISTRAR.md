---
name: REGISTRAR
parameters:
  - credEntry
parameter_types:
  credEntry: string?
---

`REGISTRAR(credEntry)` selects the registrar using an entry name from creds.json.

No `NewRegistrar()` declaration is needed.

`REGISTRAR("none")` is the  selects the registrar using an entry name from creds.json.

`REGISTRAR()` is an alias for `REGISTRAR("none")`, which disables registrar management.

`REGISTRAR()` must immediately follow the domain name in `D()`.

`REGISTRAR()` should never appear in `D_EXTEND()`.

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

`REGISTRAR()` may appear in `DEFAULTS()`, but setting a `REGISTRAR()` in a `D()` will override it.

```javascript
DEFAULTS(
    REGISTRAR(),
    SERVICE("bind", 0),
);

D("example.com", A("@", "192.0.2.1"));
D("example.net", REGISTRAR("gandi_main"), A("@", "192.0.2.2"));
```

The legacy [NewRegistrar](../top-level-functions/NewRegistrar.md) helper remains
supported. See the [conversion guide](../../getting-started/converting-dnsconfig.md).
