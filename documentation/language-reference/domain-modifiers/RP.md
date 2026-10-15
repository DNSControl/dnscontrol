---
name: RP
parameters:
  - name
  - mbox
  - txt
  - modifiers...
parameter_types:
  name: string
  mbox: string
  txt: string
  "modifiers...": RecordModifier[]
---

`RP` adds an [Responsible Person record](https://www.rfc-editor.org/rfc/rfc1183) to a domain.

An RP record contains contact details for the domain. Usually an email address with the `@` replaced by a `.`.

{% hint style="warning" %}
The RP implementation in DNSControl is still experimental and may change.
{% endhint %}

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  RP("@", "user.example.com.", "example.com."),
);
```
{% endcode %}
