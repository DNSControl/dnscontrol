DNSControl's easyname provider supports being a Registrar. Support for being a DNS Provider is not included, but could be added in the future.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `EASYNAME`
along with [API-Access](https://my.easyname.com/en/account/api) information

Example:

{% code title="creds.json" %}
```json
{
  "easyname": {
    "TYPE": "EASYNAME",
    "apikey": "API Key",
    "authsalt": "API Authentication Salt",
    "email": "example@example.com",
    "signsalt": "API Signing Salt",
    "userid": 12345
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to easyname.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_EASYNAME = NewRegistrar("easyname");

D("example.com", REG_EASYNAME,
  NAMESERVER("ns1.example.com."),
  NAMESERVER("ns2.example.com."),
);
```
{% endcode %}

## Activation

You must enable API-Access for your account.

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
