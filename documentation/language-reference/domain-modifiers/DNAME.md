---
name: DNAME
parameters:
  - name
  - target
  - modifiers...
parameter_types:
  name: string
  target: string
  "modifiers...": RecordModifier[]
---

`DNAME` adds a [Delegation name record](https://www.rfc-editor.org/rfc/rfc6672) to the domain.

Target should be a string.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  DNAME("sub", "example.net."),
);
```
{% endcode %}
