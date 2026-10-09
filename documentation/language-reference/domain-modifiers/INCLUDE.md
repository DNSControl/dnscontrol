---
name: INCLUDE
parameters:
  - domain
parameter_types:
  domain: string
---

Includes all records from a given domain


{% code title="dnsconfig.js" %}
```javascript
D("example.com!external", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  A("test", "8.8.8.8"),
);

D("example.com!internal", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  INCLUDE("example.com!external"),
  A("home", "127.0.0.1"),
);
```
{% endcode %}
