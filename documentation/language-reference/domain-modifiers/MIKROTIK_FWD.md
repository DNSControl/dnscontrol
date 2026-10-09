---
name: MIKROTIK_FWD
parameters:
  - name
  - target
  - modifiers...
parameter_types:
  name: string
  target: string
  "modifiers...": RecordModifier[]
provider: MIKROTIK
---

`MIKROTIK_FWD` creates a RouterOS FWD (conditional DNS forwarding) static entry. These records instruct the MikroTik router to forward DNS queries matching the name to a specified upstream server, optionally populating a RouterOS address list with resolved addresses.

The `target` can be an IP address (e.g. `8.8.8.8`) or the name of a [`MIKROTIK_FORWARDER`](MIKROTIK_FORWARDER.md) entry (e.g. `my-upstream`).

See the [MikroTik RouterOS provider page](../../provider/mikrotik.md) for full configuration details.

Metadata keys supported:

| Key               | Description                                                        |
|-------------------|--------------------------------------------------------------------|
| `match_subdomain` | Set to `"true"` to also match subdomains of the name.              |
| `regexp`          | RouterOS regexp for query matching.                                |
| `address_list`    | RouterOS address list to populate with resolved addresses.         |
| `comment`         | Comment stored on the RouterOS record.                             |

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
    // Forward all queries for example.com and subdomains to 8.8.8.8,
    // add resolved addresses to the "vpn-list" address list.
    MIKROTIK_FWD("@", "8.8.8.8", {match_subdomain: "true", address_list: "vpn-list"}),

    // Forward internal.example.com to a named forwarder entry.
    MIKROTIK_FWD("internal", "corp-dns", {match_subdomain: "true"}),
);
```
{% endcode %}
