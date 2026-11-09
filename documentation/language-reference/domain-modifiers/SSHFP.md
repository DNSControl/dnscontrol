---
name: SSHFP
parameters:
  - name
  - algorithm
  - type
  - value
  - modifiers...
parameter_types:
  name: string
  algorithm: 0 | 1 | 2 | 3 | 4
  type: 0 | 1 | 2
  value: string
  "modifiers...": RecordModifier[]
---

`SSHFP` adds a [SSH Public Key Fingerprint record](https://www.rfc-editor.org/rfc/rfc4255) to the domain. The record contains a fingerprint of a SSH server which can be validated before SSH clients are establishing the connection.

**Algorithm** (type of the key)

| ID | Algorithm |
|----|-----------|
| 0  | reserved  |
| 1  | RSA       |
| 2  | DSA       |
| 3  | ECDSA     |
| 4  | ED25519   |

**Type** (fingerprint format)

| ID | Algorithm |
|----|-----------|
| 0  | reserved  |
| 1  | SHA-1     |
| 2  | SHA-256   |

`value` is the fingerprint as a string.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  SSHFP("@", 1, 1, "00yourAmazingFingerprint00"),
);
```
{% endcode %}
