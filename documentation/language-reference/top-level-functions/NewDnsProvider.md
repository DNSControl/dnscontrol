---
name: NewDnsProvider
parameters:
  - name
  - meta
parameter_types:
  name: string
  meta: object?
return: string
---

NewDnsProvider activates a DNS Service Provider (DSP) specified in `creds.json`.
A DSP stores a DNS zone's records and provides DNS service for the zone (i.e.
answers on port 53 to queries related to the zone).

* `name` must match the name of an entry in `creds.json`.
* `type` is deprecated. The provider type is read from the `TYPE` field in `creds.json`.
* `meta` is a way to send additional parameters to the provider.  It is optional and only certain providers use it.  See the [individual provider docs](../../provider/index.md) for details.

This function will return an opaque string that should be assigned to a variable name for use in [D](D.md) directives.

{% code title="dnsconfig.js" %}
```javascript
var REG_MYNDC = NewRegistrar("mynamedotcom");
var DNS_MYAWS = NewDnsProvider("myaws");

D("example.com", REG_MYNDC, DnsProvider(DNS_MYAWS),
  A("@","1.2.3.4"),
);
```
{% endcode %}
