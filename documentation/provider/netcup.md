## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `NETCUP`
along with your [api key, password and your customer number](https://www.netcup-wiki.de/wiki/CCP_API#Authentifizierung).

Example:

{% code title="creds.json" %}
```json
{
  "netcup": {
    "TYPE": "NETCUP",
    "api-key": "abc12345",
    "api-password": "abc12345",
    "customer-number": "123456"
  }
}
```
{% endcode %}

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE("netcup"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Caveats
Netcup does not allow any TTLs to be set for individual records. Thus in
the diff/preview it will always show a TTL of 0. `NS` records are also
not currently supported.

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
  - [`A`](../language-reference/domain-modifiers/A.md) [`AAAA`](../language-reference/domain-modifiers/AAAA.md) [`CAA`](../language-reference/domain-modifiers/CAA.md) [`CNAME`](../language-reference/domain-modifiers/CNAME.md) [`MX`](../language-reference/domain-modifiers/MX.md) [`SRV`](../language-reference/domain-modifiers/SRV.md) [`TLSA`](../language-reference/domain-modifiers/TLSA.md) [`TXT`](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
