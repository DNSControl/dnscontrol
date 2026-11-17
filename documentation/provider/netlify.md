## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `NETLIFY`
along with a Netlify account personal access token. You can also optionally add an
account slug. This is _typically_ your username on Netlify.

Examples:

{% code title="creds.json" %}
```json
{
  "netlify": {
    "TYPE": "NETLIFY",
    "token": "your-netlify-account-access-token",
    "slug": "account-slug" // this is optional
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to Netlify.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("netlify"), SERVICE("netlify"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
DNSControl depends on a Netlify account personal access token.

## Caveats
Empty MX records are not supported.

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
