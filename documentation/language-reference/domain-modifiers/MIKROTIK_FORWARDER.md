---
name: MIKROTIK_FORWARDER
parameters:
  - name
  - dns_servers
  - modifiers...
parameter_types:
  name: string
  dns_servers: string
  "modifiers...": RecordModifier[]
provider: MIKROTIK
---

`MIKROTIK_FORWARDER` manages a RouterOS DNS forwarder entry (`/ip/dns/forwarders`). The `name` parameter can be a domain name (e.g. `corp.example.com`) or an arbitrary alias (e.g. `my-upstream`). These named entries can then be referenced as the target of [`MIKROTIK_FWD`](MIKROTIK_FWD.md) records.

Forwarder records must be placed in the synthetic zone `_forwarders.mikrotik`. This zone should appear **before** any zones that reference its entries by name in `dnsconfig.js` to ensure proper creation order.

See the [MikroTik RouterOS provider page](../../provider/mikrotik.md) for full configuration details.

Metadata keys supported:

| Key                | Description                                        |
|--------------------|----------------------------------------------------|
| `doh_servers`      | DoH server URLs for this forwarder.                |
| `verify_doh_cert`  | Set to `"true"` to verify the DoH certificate.     |
| `comment`          | Comment stored on the RouterOS forwarder entry.    |

{% code title="dnsconfig.js" %}
```javascript
D("_forwarders.mikrotik", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
    // Domain-based forwarder: forward corp.example.com to internal DNS servers.
    MIKROTIK_FORWARDER("corp.example.com", "10.0.0.53,10.0.0.54"),

    // Alias-based forwarder with DoH.
    MIKROTIK_FORWARDER("doh-upstream", "1.1.1.1", {doh_servers: "https://cloudflare-dns.com/dns-query", verify_doh_cert: "true"}),
);

// Then reference the alias in a FWD record:
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
    MIKROTIK_FWD("@", "doh-upstream", {match_subdomain: "true"}),
);
```
{% endcode %}
