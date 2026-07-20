---
name: DOMAIN_ELSEWHERE
parameters:
  - name
  - registrar
  - nameserver_names
parameter_types:
  name: string
  registrar: string
  nameserver_names: string[]
---

`DOMAIN_ELSEWHERE()` is a helper macro that lets you easily indicate that a domain's zones are managed elsewhere. That is, it permits you easily delegate a domain to a hard-coded list of DNS servers.

`DOMAIN_ELSEWHERE` is useful when you control a domain's registrar but not the DNS servers. For example, suppose you own a domain but the DNS servers are run by someone else, perhaps a SaaS product you've subscribed to or a DNS server that is run by your brother-in-law who doesn't trust you with the API keys that would let you maintain the domain using DNSControl. You need an easy way to point (delegate) the domain at a specific list of DNS servers.

For example these two statements are equivalent:

{% code title="dnsconfig.js" %}
```javascript
DOMAIN_ELSEWHERE("example.com", REG_MY_PROVIDER, ["ns1.foo.com", "ns2.foo.com"]);
```
{% endcode %}

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    NO_PURGE,
    NAMESERVER("ns1.foo.com"),
    NAMESERVER("ns2.foo.com"),
);
```
{% endcode %}

{% hint style="info" %}
**NOTE**: The [`NO_PURGE`](../domain-modifiers/NO_PURGE.md) is used out of abundance of caution but since no
`DnsProvider()` statements exist, no updates would be performed.
{% endhint %}
