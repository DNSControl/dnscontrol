---
name: HTTPS
parameters:
  - name
  - priority
  - target
  - params
  - modifiers...
parameter_types:
  name: string
  priority: number
  target: string
  params: string
  "modifiers...": RecordModifier[]
---

HTTPS adds an HTTPS record to a domain. The name should be the relative label for the record. Use `@` for the domain apex. The HTTPS record is a special form of the SVCB resource record.

The priority must be a positive number, the address should be an ip address, either a string, or a numeric value obtained via [IP](../top-level-functions/IP.md).

The params may be configured to specify the `alpn`, `ipv4hint`, `ipv6hint`, `ech` or `port` setting. Several params may be joined by a space. Not existing params may be specified as an empty string `""`

Modifiers can be any number of [record modifiers](https://docs.dnscontrol.org/language-reference/record-modifiers) or JSON objects, which will be merged into the record's metadata.

If you set the parameter `ech` to the special value `IGNORE`, DNSControl will ignore the contents of that parameter when updating a zone.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  HTTPS("@", 1, ".", "ipv4hint=123.123.123.123 alpn=h3,h2 port=443"),
  HTTPS("@", 1, "test.com", ""),
);
```
{% endcode %}
