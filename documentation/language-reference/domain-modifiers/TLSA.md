---
name: TLSA
parameters:
  - name
  - usage
  - selector
  - type
  - certificate
  - modifiers...
parameter_types:
  name: string
  usage: number
  selector: number
  type: number
  certificate: string
  "modifiers...": RecordModifier[]
---

`TLSA` adds a [TLSA certificate association record](https://www.rfc-editor.org/rfc/rfc6698) to a domain. The name should be the relative label for the record.

Usage, selector, and type are ints.

Certificate is a hex string.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  // Create TLSA record for certificate used on TCP port 443
  TLSA("_443._tcp", 3, 1, 1, "abcdef0"),
);
```
{% endcode %}
