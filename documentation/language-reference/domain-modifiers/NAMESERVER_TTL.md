---
name: NAMESERVER_TTL
parameters:
  - ttl
parameter_types:
  ttl: Duration
---

NAMESERVER_TTL sets the TTL on the domain apex NS RRs defined by [`NAMESERVER`](NAMESERVER.md).

The value can be an integer or a string. See [`TTL`](../record-modifiers/TTL.md) for examples.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  NAMESERVER_TTL("2d"),
  NAMESERVER("ns"),
);
```
{% endcode %}

Use `NAMESERVER_TTL("3600"),` or `NAMESERVER_TTL("1h"),` for a 1h default TTL for all subsequent `NS` entries:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  DefaultTTL("4h"),
  NAMESERVER_TTL("3600"),
  NAMESERVER("ns1.provider.com."), //inherits NAMESERVER_TTL
  NAMESERVER("ns2.provider.com."), //inherits NAMESERVER_TTL
  A("@","1.2.3.4"), // inherits DefaultTTL
  A("foo", "2.3.4.5", TTL(600)), // overrides DefaultTTL for this record only
);
```
{% endcode %}

To apply a default TTL to all other record types, see [`DefaultTTL`](../domain-modifiers/DefaultTTL.md)
