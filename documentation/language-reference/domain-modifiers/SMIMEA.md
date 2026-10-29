---
name: SMIMEA
parameters:
  - name
  - usage
  - selector
  - type
  - certificate
  - modifiers...
parameter_types:
  name: string
  usage: number
  selector: number
  type: number
  certificate: string
  "modifiers...": RecordModifier[]
---

`SMIMEA` adds an [S/MIME cert association record](https://www.rfc-editor.org/rfc/rfc8162) to a domain. The name should be the hashed and stripped local part of the e-mail.

To create the name, you can the following command:

```bash
# For the e-mail bosun@bosun.org run:
echo -n "bosun" | sha256sum | awk '{print $1}' | cut -c1-56
# f10e7de079689f55c0cdd6782e4dd1448c84006962a4bd832e8eff73
```

Usage, selector, and type are ints.

Certificate is a hex string.

To create the string for the type 0, you can run this command with your S/MIME certificate:

```bash
openssl x509 -in smime-cert.pem -outform DER | xxd -p -c 10000
```

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  // Create SMIMEA record for certificate for the name bosun
  SMIMEA("f10e7de079689f55c0cdd6782e4dd1448c84006962a4bd832e8eff73", 3, 0, 0, "30820353308202f8a003020102..."),
);
```
{% endcode %}
