---
name: CAA
parameters:
  - name
  - tag
  - value
  - modifiers...
parameter_types:
  name: string
  tag: '"issue" | "issuewild" | "iodef" | "contactemail" | "contactphone" | "issuemail" | "issuevmc"'
  value: string
  "modifiers...": RecordModifier[]
---

`CAA` adds a [Certification Authority Authorization record](https://www.rfc-editor.org/rfc/rfc8659) to a domain. The name should be the relative label for the record. Use `@` for the domain apex.

Tag can be one of
1. `"issue"`
2. `"issuewild"`
3. `"iodef"`
4. `"contactemail"`
5. `"contactphone"`
6. `"issuemail"`
7. `"issuevmc"`

Value is a string. The format of the contents is different depending on the tag. DNSControl will handle any escaping or quoting required, similar to TXT records. For example use `CAA("@", "issue", "letsencrypt.org")` rather than `CAA("@", "issue", "\"letsencrypt.org\"")`.

Flags are controlled by modifier:
- `CAA_CRITICAL`: Issuer critical flag. CA that does not understand this tag will refuse to issue certificate for this domain.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  // Allow letsencrypt to issue certificate for this domain
  CAA("@", "issue", "letsencrypt.org"),
  // Allow no CA to issue wildcard certificate for this domain
  CAA("@", "issuewild", ";"),
  // Report all violation to test@example.com. If CA does not support
  // this record then refuse to issue any certificate
  CAA("@", "iodef", "mailto:test@example.com", CAA_CRITICAL),
);
```
{% endcode %}

DNSControl contains a [`CAA_BUILDER`](CAA_BUILDER.md) which can be used to simply create `CAA()` records for your domains. Instead of creating each CAA record individually, you can simply configure your report mail address, the authorized certificate authorities and the builder cares about the rest.
