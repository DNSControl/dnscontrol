---
name: REV
parameters:
  - address
parameter_types:
  address: string
ts_return: string
---

`REV` returns the reverse lookup domain for an IP network. For example `REV("1.2.3.0/24")` returns `3.2.1.in-addr.arpa.` and `REV("2001:db8:302::/48")` returns `2.0.3.0.8.b.d.0.1.0.0.2.ip6.arpa.`.

`REV()` is commonly used with the [`D()`](D.md) functions to create reverse DNS lookup zones.

These two are equivalent:

{% code title="dnsconfig.js" %}
```javascript
D("3.2.1.in-addr.arpa", ...
```
{% endcode %}

{% code title="dnsconfig.js" %}
```javascript
D(REV("1.2.3.0/24", ...
```
{% endcode %}

The latter is easier to type and less error-prone.

If the address does not include a "/" then `REV()` assumes /32 for IPv4 addresses and /128 for IPv6 addresses.

# RFC compliance

`REV()` implements both RFC 2317 and the newer RFC 4183. The `REVCOMPAT()` function selects which mode is used. If `REVCOMPAT()` is not called, a default is selected for you.  The default will change to RFC 4183 in DNSControl v5.0.

See [`REVCOMPAT()`](REVCOMPAT.md) for details.

# Host bits

v4.x: The host bits (the ones outside the netmask) must be zeros. They are not zeroed out automatically. Thus, `REV("1.2.3.4/24")` is an error.

v5.0 and later: The host bits (the ones outside the netmask) are ignored.  Thus `REV("1.2.3.4/24")` and `REV("1.2.3.0/24")` are equivalent.

# Examples

Here's an example reverse lookup domain:

{% code title="dnsconfig.js" %}
```javascript
D(REV("1.2.3.0/24"), REGISTRAR, DnsProvider(BIND),
  PTR("1", "foo.example.com."),
  PTR("2", "bar.example.com."),
  PTR("3", "baz.example.com."),
  // If the first parameter is an IP address, DNSControl automatically calls REV() for you.
  PTR("1.2.3.10", "ten.example.com."),
);

D(REV("2001:db8:302::/48"), REGISTRAR, DnsProvider(BIND),
  PTR("1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0", "foo.example.com."),  // 2001:db8:302::1
  // If the first parameter is an IP address, DNSControl automatically calls REV() for you.
  PTR("2001:db8:302::2", "two.example.com."),                          // 2.0.0...
  PTR("2001:db8:302::3", "three.example.com."),                        // 3.0.0...
);
```
{% endcode %}

# Automatic forward and reverse record generation

DNSControl does not automatically generate forward and reverse lookups. However it is possible to write a macro that does this.  See [`PTR()`](../domain-modifiers/PTR.md)   for an example.
