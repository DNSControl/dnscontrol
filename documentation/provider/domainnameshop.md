## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `DOMAINNAMESHOP`
along with your [Domainnameshop Token and Secret](https://www.domeneshop.no/admin?view=api).

Example:

{% code title="creds.json" %}
```json
{
  "mydomainnameshop": {
    "TYPE": "DOMAINNAMESHOP",
    "token": "your-domainnameshop-token",
    "secret": "your-domainnameshop-secret"
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to Domainnameshop.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE("mydomainnameshop"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
[Create API Token and secret](https://www.domeneshop.no/admin?view=api)

## Limitations

- Domainnameshop DNS only supports TTLs which are a multiple of 60.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Record types
  - [`A`](../language-reference/domain-modifiers/A.md) [`AAAA`](../language-reference/domain-modifiers/AAAA.md) [`CAA`](../language-reference/domain-modifiers/CAA.md) [`CNAME`](../language-reference/domain-modifiers/CNAME.md) [`MX`](../language-reference/domain-modifiers/MX.md) [`NS`](../language-reference/domain-modifiers/NS.md) [`SRV`](../language-reference/domain-modifiers/SRV.md) [`TXT`](../language-reference/domain-modifiers/TXT.md)
    - Unimplemented: [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md) (Needs custom implementation)
    - Unimplemented: [`DS`](../language-reference/domain-modifiers/DS.md)
    - Unimplemented: [`TLSA`](../language-reference/domain-modifiers/TLSA.md) (Has support but no documentation. Needs to be investigated.)
    - Not supported: [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md) (According to Domainnameshop this will probably never be supported)
    - Not supported: [`PTR`](../language-reference/domain-modifiers/PTR.md) (According to Domainnameshop this will probably never be supported)
    - Not supported: [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md) (Might be supported in the future)
<!-- provider-features-end -->
