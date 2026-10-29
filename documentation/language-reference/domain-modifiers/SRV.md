---
name: SRV
parameters:
  - name
  - priority
  - weight
  - port
  - target
  - modifiers...
parameter_types:
  name: string
  priority: number
  weight: number
  port: number
  target: string
  "modifiers...": RecordModifier[]
---

`SRV` adds a [Service locator record](https://www.rfc-editor.org/rfc/rfc2782) to a domain. The name should be the relative label for the record.

Priority, weight, and port are ints.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  // Create SRV records for a a SIP service:
  //               pr  w   port, target
  SRV("_sip._tcp", 10, 60, 5060, "bigbox.example.com."),
  SRV("_sip._tcp", 10, 20, 5060, "smallbox1.example.com."),
);
```
{% endcode %}
