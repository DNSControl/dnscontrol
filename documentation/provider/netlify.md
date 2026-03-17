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
var REG_NETLIFY = NewRegistrar("netlify");
var DSP_NETLIFY = NewDnsProvider("netlify");

D("example.com", REG_NETLIFY, DnsProvider(DSP_NETLIFY),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
DNSControl depends on a Netlify account personal access token.

## Caveats
Empty MX records are not supported.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
