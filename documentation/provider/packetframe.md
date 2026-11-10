## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `PACKETFRAME`
along with your Packetframe Token which can be extracted from the `token` cookie on packetframe.com

Example:

{% code title="creds.json" %}
```json
{
  "packetframe": {
    "TYPE": "PACKETFRAME",
    "token": "your-packetframe-token"
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to Packetframe.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE("packetframe"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Record types
  - [`A`](../language-reference/domain-modifiers/A.md) [`AAAA`](../language-reference/domain-modifiers/AAAA.md) [`CNAME`](../language-reference/domain-modifiers/CNAME.md) [`MX`](../language-reference/domain-modifiers/MX.md) [`NS`](../language-reference/domain-modifiers/NS.md) [`PTR`](../language-reference/domain-modifiers/PTR.md) [`SRV`](../language-reference/domain-modifiers/SRV.md) [`TXT`](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
