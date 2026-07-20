---
name: NewRegistrar
parameters:
  - name
  - type
  - meta
parameter_types:
  name: string
  type: string?
  meta: object?
return: string
---

NewRegistrar activates a Registrar Provider specified in `creds.json`.
A registrar maintains the domain's registration and delegation (i.e. the
nameservers for the domain).  DNSControl only manages the delegation.

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
