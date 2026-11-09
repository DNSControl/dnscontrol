---
name: DKIM_BUILDER
parameters:
  - selector
  - pubkey
  - label
  - version
  - hashtypes
  - keytype
  - note
  - servicetypes
  - flags
  - ttl
parameters_object: true
parameter_types:
  selector: string
  pubkey: string?
  label: string?
  version: string?
  hashtypes: string|string[]?
  keytype: string?
  note: string?
  servicetypes: string|string[]?
  flags: string|string[]?
  ttl: Duration?
---

DNSControl contains a `DKIM_BUILDER` helper function that generates DKIM DNS TXT records according to RFC 6376 (DomainKeys Identified Mail) and its updates.

## Examples

### Simple example

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  DKIM_BUILDER({
    selector: "s1",
    pubkey: "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDC5/z4L"
  }),
);
```
{% endcode %}

This yield the following record:

```text
s1._domainkey   IN  TXT "v=DKIM1; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDC5/z4L"
```

### Advanced example

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  DKIM_BUILDER({
    selector: "k2",
    pubkey: "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDC5/z4L",
    label: "subdomain",
    version: "DKIM1",
    hashtypes: ["sha1", "sha256"],
    keytype: "rsa",
    note: "some human-readable notes",
    servicetypes: ["email"],
    flags: ["y", "s"],
    ttl: 150
  }),
);
```
{% endcode %}

This yields the following record:

```text
k2._domainkey.subdomain   IN  TXT "v=DKIM1; h=sha1:sha256; k=rsa; n=some=20human-readable=20notes; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDC5/z4L; s=email; t=y:s" ttl=150
```

## Parameters

* `selector` (string, required): The selector subdividing the namespace for the domain.
* `pubkey` (string, optional): The base64-encoded public key (RSA or Ed25519). Default: empty (key revocation or non-sending domain).
* `label` (string, optional): The DNS label for the DKIM record. Default: `@`.
* `version` (string, optional): DKIM version. Maps to the `v=` tag. Default: `DKIM1` (currently the only supported value).
* `hashtypes` (array, optional): Acceptable hash algorithms for signing. Maps to the `h=` tag.
  * Supported values for RSA key:
    * `sha1`
    * `sha256`
  * Supported values for Ed25519 key:
    * `sha256`
* `keytype` (string, optional): Key algorithm type. Maps to the `k=` tag. Default: `rsa`. Supported values:
   * `rsa`
   * `ed25519`
* `note` (string, optional): Human-readable notes intended for administrators. Pass normal text here; DKIM-Quoted-Printable encoding will be applied automatically. Maps to the `n=` tag.
* `servicetypes` (array, optional): Service types using this key. Maps to the `s=` tag. Supported values:
  * `*`: explicitly allows all service types
  * `email`: restricts key to email service only
* `flags` (array, optional): Flags to modify the interpretation of the selector. Maps to the `t=` tag. Supported values:
  * `y`: Testing mode.
  * `s`: Subdomain restriction.
* `ttl` (number, optional): DNS TTL value in seconds

## Related RFCs

* RFC 6376: DomainKeys Identified Mail (DKIM) Signatures
* RFC 8301: Cryptographic Algorithm and Key Usage Update to DKIM
* RFC 8463: A New Cryptographic Signature Method for DKIM (Ed25519)
