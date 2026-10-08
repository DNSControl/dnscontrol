---
name: DefaultTTL
parameters:
  - ttl
parameter_types:
  ttl: Duration
---

DefaultTTL sets the Time To Live (TTL) for all subsequent records following it in a domain that do not explicitly set one with [`TTL`](../record-modifiers/TTL.md). If neither `DefaultTTL` or `TTL` exist for a record,
the record will inherit the DNSControl global internal default of 300 seconds. See also [`DEFAULTS`](../top-level-functions/DEFAULTS.md) to override the internal defaults.

NS records are currently a special case, and do not inherit from `DefaultTTL`. See [`NAMESERVER_TTL`](../domain-modifiers/NAMESERVER_TTL.md) to set a default TTL for all NS records.


{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  DefaultTTL("4h"),
  A("@","1.2.3.4"), // uses default
  A("foo", "2.3.4.5", TTL(600)), // overrides default
);
```
{% endcode %}

The DefaultTTL duration is the same format as [`TTL`](../record-modifiers/TTL.md), an integer number of seconds
or a string with a unit such as `"4d"`.
