---
name: PORKBUN_URLFWD
parameters:
  - name
  - target
  - modifiers...
provider: PORKBUN
parameter_types:
  name: string
  target: string
  "modifiers...": RecordModifier[]
---

{% hint style="warning" %}
**DEPRECATED**: This record type is deprecated. Please use `URL` (for temporary redirects) or `URL301` (for permanent redirects) instead. PORKBUN_URLFWD will continue to work but is no longer recommended for new configurations.
{% endhint %}

`PORKBUN_URLFWD` is a [Porkbun](../../provider/porkbun.md)-specific feature that maps to Porkbun's URL forwarding feature, which creates HTTP 301 (permanent) or 302 (temporary) redirects.


{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
    PORKBUN_URLFWD("urlfwd1", "http://example.com"),
    PORKBUN_URLFWD("urlfwd2", "http://example.org", {type: "permanent", includePath: "yes", wildcard: "no"})
);
```
{% endcode %}

The fields are:
* name: the record name
* target: where you'd like to forward the domain to
* type: valid types are: `temporary` (302 / 307) or `permanent` (301), default to `temporary`
* includePath: whether to include the URI path in the redirection. Valid options are `yes` or `no`, default to `no`
* wildcard: forward all subdomains of the domain. Valid options are `yes` or `no`, default to `yes`
