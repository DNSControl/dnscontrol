This is the provider for [Infomaniak](https://www.infomaniak.com/).

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `INFOMANIAK` along with a Infomaniak account personal access token.

Examples:

{% code title="creds.json" %}
```json
{
  "infomaniak": {
    "TYPE": "INFOMANIAK",
    "token": "your-infomaniak-account-access-token",
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to Infomaniak.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE("infomaniak"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
DNSControl depends on a Infomaniak account personal access token.

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [DS](../language-reference/domain-modifiers/DS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
