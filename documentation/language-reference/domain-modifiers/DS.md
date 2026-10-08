---
name: DS
parameters:
  - name
  - keytag
  - algorithm
  - digesttype
  - digest
  - modifiers...
parameter_types:
  name: string
  keytag: number
  algorithm: number
  digesttype: number
  digest: string
  "modifiers...": RecordModifier[]
---

`DS` adds a [Delegation signer record](https://www.rfc-editor.org/rfc/rfc4034) to the domain.

Key Tag should be a number.

Algorithm should be a number.

Digest Type must be a number.

Digest must be a string.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  DS("example.com", 2371, 13, 2, "ABCDEF"),
);
```
{% endcode %}
