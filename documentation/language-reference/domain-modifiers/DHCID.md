---
name: DHCID
parameters:
  - name
  - digest
  - modifiers...
parameter_types:
  name: string
  digest: string
  "modifiers...": RecordModifier[]
---

`DHCID` adds a [DHCP identifier record](https://www.rfc-editor.org/rfc/rfc4701) to the domain.

Digest should be a string.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  DHCID("example.com", "ABCDEFG"),
);
```
{% endcode %}
