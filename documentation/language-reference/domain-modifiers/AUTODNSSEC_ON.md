---
name: AUTODNSSEC_ON
---

`AUTODNSSEC_ON` tells the provider to **enable** AutoDNSSEC.

[`AUTODNSSEC_OFF`](AUTODNSSEC_OFF.md) tells the provider to **disable** AutoDNSSEC.

AutoDNSSEC is a feature where a DNS provider can automatically manage DNSSEC for a domain. Not all providers support this.

At this time, `AUTODNSSEC_ON` takes no parameters.  There is no ability to tune what the DNS provider sets, no algorithm choice.  We simply ask that they follow their defaults when enabling a no-fuss DNSSEC data model.

{% hint style="info" %}
**NOTE**: No parenthesis should follow these keywords.  That is, the
correct syntax is `AUTODNSSEC_ON` not `AUTODNSSEC_ON()`
{% endhint %}

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  AUTODNSSEC_ON,  // Enable AutoDNSSEC.
  A("@", "10.1.1.1"),
);

D("insecure.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  AUTODNSSEC_OFF,  // Disable AutoDNSSEC.
  A("@", "10.2.2.2"),
);
```
{% endcode %}

If neither `AUTODNSSEC_ON` or `AUTODNSSEC_OFF` is specified for a domain no changes will be requested.
