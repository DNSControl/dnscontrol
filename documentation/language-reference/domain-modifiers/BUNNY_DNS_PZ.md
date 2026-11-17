---
name: BUNNY_DNS_PZ
parameters:
  - name
  - pullZoneId
  - modifiers...
parameter_types:
  name: string
  pullZoneId: number
  "modifiers...": RecordModifier[]
provider: BUNNY_DNS
---

`BUNNY_DNS_PZ` creates a Bunny.net DNS **Pull Zone** record, which points the record's name at a Bunny Pull Zone. The zone then serves the pull zone's cached content.

This record type is specific to the [Bunny DNS provider](../../provider/bunnydns.md); it cannot be used with any other DNS provider.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("bunny_dns"),
    BUNNY_DNS_PZ("@", 12345),
);
```
{% endcode %}

The `pullZoneId` parameter is the numeric ID of the Pull Zone, as shown in the Bunny.net dashboard.

Bunny DNS offers other custom record types (such as Script and Flatten) that DNSControl does not manage; DNSControl leaves them as-is.
