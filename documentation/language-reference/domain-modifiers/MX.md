---
name: MX
parameters:
  - name
  - priority
  - target
  - modifiers...
parameter_types:
  name: string
  priority: number
  target: string
  "modifiers...": RecordModifier[]
---

`MX` adds a [Mail exchange record](https://www.rfc-editor.org/rfc/rfc1035) to the domain.

Priority should be a number.

Target should be a string representing the MX target. If it is a single label we will assume it is a relative name on the current domain. If it contains *any* dots, it should be a fully qualified domain name, ending with a `.`.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  MX("@", 5, "mail"), // mx example.com -> mail.example.com
  MX("sub", 10, "mail.foo.com."),
);
```
{% endcode %}
