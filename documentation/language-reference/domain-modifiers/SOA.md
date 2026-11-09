---
name: SOA
parameters:
  - name
  - ns
  - mbox
  - refresh
  - retry
  - expire
  - minttl
  - modifiers...
parameter_types:
  name: string
  ns: string
  mbox: string
  refresh: number
  retry: number
  expire: number
  minttl: number
  "modifiers...": RecordModifier[]
---

`SOA` adds a [Start of Authority record](https://www.rfc-editor.org/rfc/rfc1035) to a domain. The name should be `@`.  ns and mbox are strings. The other fields are unsigned 32-bit ints.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  SOA("@", "ns3.example.com.", "hostmaster.example.com.", 3600, 600, 604800, 1440),
);
```
{% endcode %}

## Notes
* The serial number is managed automatically.  It isn't even a field in `SOA`.
* Most providers automatically generate SOA records.  They will ignore any `SOA` statements.
* The mbox field should not be set to a real email address unless you love spam and hate your privacy.

There is more info about `SOA` in the documentation for the [BIND provider](../../provider/bind.md).
