---
name: HEDNS_DYNAMIC_ON
parameters: []
ts_return: RecordModifier
provider: HEDNS
---

`HEDNS_DYNAMIC_ON` enables [Dynamic DNS](https://dns.he.net/) on a record managed by the Hurricane Electric DNS provider. When enabled, HE DNS assigns a DDNS key to the record that can be used with the HE DDNS update API (`https://dyn.dns.he.net/nic/update`).

If a record is already dynamic, its dynamic state is preserved across modifications even without explicitly specifying this modifier.

To set a specific DDNS key, use [`HEDNS_DDNS_KEY()`](HEDNS_DDNS_KEY.md) instead.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("none"), SERVICE("hedns"),
    A("dyn", "0.0.0.0", HEDNS_DYNAMIC_ON),
    AAAA("dyn6", "::1", HEDNS_DYNAMIC_ON),
);
```
{% endcode %}
