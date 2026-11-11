DNSControl's Dynadot provider supports being a Registrar. Support for being a DNS Provider is not included, but could be added in the future.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `DYNADOT`
along with `key` from the [Dynadot API](https://www.dynadot.com/account/domain/setting/api.html).

Example:

{% code title="creds.json" %}
```json
{
  "dynadot": {
    "TYPE": "DYNADOT",
    "key": "API Key",
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to Dynadot.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
DOMAIN_ELSEWHERE("example.com", REGISTRAR("dynadot"), [
    "ns1.example.net.",
    "ns2.example.net.",
    "ns3.example.net.",
]);
```
{% endcode %}

## Activation

You must [enable the Dynadot API](https://www.dynadot.com/account/domain/setting/api.html) for your account and whitelist the IP address of the machine that will run DNSControl.

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ❌
  - Registrar: ✅
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
<!-- provider-features-end -->
