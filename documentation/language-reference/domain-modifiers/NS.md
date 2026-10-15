---
name: NS
parameters:
  - name
  - target
  - modifiers...
parameter_types:
  name: string
  target: string
  "modifiers...": RecordModifier[]
---

`NS` adds a [Name server record](https://www.rfc-editor.org/rfc/rfc1035) to the domain. The name should be the relative label for the domain.

The name may not be `@` (the bare domain), as that is controlled via [`NAMESERVER()`](NAMESERVER.md).
The difference between `NS()` and [`NAMESERVER()`](NAMESERVER.md) is explained in the [`NAMESERVER()` description](NAMESERVER.md).


Target should be a string representing the NS target. If it is a single label we will assume it is a relative name on the current domain. If it contains *any* dots, it should be a fully qualified domain name, ending with a `.`.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  NS("foo", "ns1.example2.com."), // Delegate ".foo.example.com" zone to another server.
  NS("foo", "ns2.example2.com."), // Delegate ".foo.example.com" zone to another server.
  A("ns1.example2.com", "10.10.10.10"), // Glue records
  A("ns2.example2.com", "10.10.10.20"), // Glue records
);
```
{% endcode %}
