---
name: HEDNS_DDNS_KEY
parameters:
  - key
parameter_types:
  key: string
ts_return: RecordModifier
provider: HEDNS
---

`HEDNS_DDNS_KEY` enables Dynamic DNS on a record managed by the Hurricane Electric DNS provider and sets a specific DDNS key (token). This implies [`HEDNS_DYNAMIC_ON`](HEDNS_DYNAMIC_ON.md).

The DDNS key can then be used with the HE DDNS update API (`https://dyn.dns.he.net/nic/update`) to update the record's value.

**Note:** DDNS keys are **write-only**. dnscontrol sets the key on the provider but cannot read back the current key. This means a key-only change (same record data, new key) will not be detected as a difference. To force an update, also change another field such as the TTL.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("none"), SERVICE("hedns"),
    A("dyn", "0.0.0.0", HEDNS_DDNS_KEY("my-secret-token")),
    AAAA("dyn6", "::1", HEDNS_DDNS_KEY("another-token")),
);
```
{% endcode %}
