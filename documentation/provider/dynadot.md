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
var REG_DYNADOT = NewRegistrar("dynadot");

DOMAIN_ELSEWHERE("example.com", REG_DYNADOT, [
    "ns1.example.net.",
    "ns2.example.net.",
    "ns3.example.net.",
]);
```
{% endcode %}

## Activation

You must [enable the Dynadot API](https://www.dynadot.com/account/domain/setting/api.html) for your account and whitelist the IP address of the machine that will run DNSControl.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ❌
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❌
  - get-zones: ❔
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❔
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❔
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ❔
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ❔
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❔
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❔
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
