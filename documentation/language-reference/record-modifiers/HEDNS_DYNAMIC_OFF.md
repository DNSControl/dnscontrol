---
name: HEDNS_DYNAMIC_OFF
parameters: []
ts_return: RecordModifier
provider: HEDNS
---

`HEDNS_DYNAMIC_OFF` explicitly disables Dynamic DNS on a record managed by the Hurricane Electric DNS provider. This will clear any DDNS key previously associated with the record.

Use this modifier when you want to ensure a record that was previously dynamic is returned to a static state.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("none"), SERVICE("hedns"),
    A("static", "5.6.7.8", HEDNS_DYNAMIC_OFF),
);
```
{% endcode %}
