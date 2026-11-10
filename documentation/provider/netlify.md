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
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Record types
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
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
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
<!-- provider-features-end -->
