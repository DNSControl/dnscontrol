---
name: ADGUARDHOME_AAAA_PASSTHROUGH
parameters:
  - source
  - destination
  - modifiers...
provider: ADGUARDHOME
parameter_types:
  source: string
  destination: string
  "modifiers...": RecordModifier[]
---

`ADGUARDHOME_AAAA_PASSTHROUGH` represents the literal 'A'. AdGuardHome uses this to passthrough
the original values of a record type.

The second argument to this record type must be empty.

See [this](https://github.com/AdguardTeam/Adguardhome/wiki/Configuration) page for
more information.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  ADGUARDHOME_AAAA_PASSTHROUGH("foo", ""),
);
```
{% endcode %}
