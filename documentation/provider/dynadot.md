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
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ❌
  - Registrar: ✅
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Record types
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ❌
  - [`CSYNC`](../language-reference/domain-modifiers/CSYNC.md): ❌
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❌
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`OPENPGPKEY`](../language-reference/domain-modifiers/OPENPGPKEY.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`RP`](../language-reference/domain-modifiers/RP.md): ❌
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ❌
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
<!-- provider-features-end -->
