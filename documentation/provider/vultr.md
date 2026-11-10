## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `VULTR`
along with a Vultr personal access token.

Example:

{% code title="creds.json" %}
```json
{
  "vultr": {
    "TYPE": "VULTR",
    "token": "your-vultr-personal-access-token"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Vultr.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("dnsimple"), SERVICE("vultr"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation

Vultr depends on a Vultr personal access token.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
- Record types
  - [`A`](../language-reference/domain-modifiers/A.md) [`AAAA`](../language-reference/domain-modifiers/AAAA.md) [`CAA`](../language-reference/domain-modifiers/CAA.md) [`CNAME`](../language-reference/domain-modifiers/CNAME.md) [`MX`](../language-reference/domain-modifiers/MX.md) [`NS`](../language-reference/domain-modifiers/NS.md) [`SRV`](../language-reference/domain-modifiers/SRV.md) [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md) [`TXT`](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
