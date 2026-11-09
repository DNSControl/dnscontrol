---
name: CNAME
parameters:
  - name
  - target
  - modifiers...
parameter_types:
  name: string
  target: string
  "modifiers...": RecordModifier[]
---

`CNAME` adds a [Canonical name record](https://www.rfc-editor.org/rfc/rfc1035) to the domain. The name should be the relative label for the domain.
Using `@` or `*` for CNAME records is not recommended, as different providers support them differently.

Target should be a string representing the CNAME target. If it is a single label we will assume it is a relative name on the current domain. If it contains *any* dots, it should be a fully qualified domain name, ending with a `.`.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  CNAME("foo", "google.com."), // foo.example.com -> google.com
  CNAME("abc", "@"), // abc.example.com -> example.com
  CNAME("def", "test"), // def.example.com -> test.example.com
);
```
{% endcode %}
