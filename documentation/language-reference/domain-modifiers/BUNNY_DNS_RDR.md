---
name: BUNNY_DNS_RDR
parameters:
  - name
  - target
  - modifiers...
parameter_types:
  name: string
  target: string
  "modifiers...": RecordModifier[]
provider: BUNNY_DNS
---

`BUNNY_DNS_RDR` creates a Bunny.net DNS **Redirect** record, which forwards requests for the record's name to a URL of your choosing.

This record type is specific to the [Bunny DNS provider](../../provider/bunnydns.md); it cannot be used with any other DNS provider.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("bunny_dns"),
    BUNNY_DNS_RDR("@", "https://foo.bar"),
);
```
{% endcode %}

The `target` parameter is the destination URL.

Bunny DNS offers other custom record types (such as Script and Flatten) that DNSControl does not manage; DNSControl leaves them as-is.
