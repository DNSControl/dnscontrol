---
name: A
parameters:
  - name
  - address
  - modifiers...
parameter_types:
  name: string
  address: string | number
  "modifiers...": RecordModifier[]
---

`A` adds an [IPv4 Address record](https://www.rfc-editor.org/rfc/rfc1035) to a domain. The name should be the relative label for the record. Use `@` for the domain apex.

The address should be an IP address, either a string, or a numeric value obtained via [IP](../top-level-functions/IP.md).

Modifiers can be any number of [record modifiers](https://docs.dnscontrol.org/language-reference/record-modifiers) or JSON objects, which will be merged into the record's metadata.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  A("@", "1.2.3.4"),
  A("foo", "2.3.4.5"),
  A("test.foo", IP("1.2.3.4"), TTL(5000)),
  A("*", "1.2.3.4", {foo: 42}),
);
```
{% endcode %}
