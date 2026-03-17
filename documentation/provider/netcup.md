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
var REG_NONE = NewRegistrar("none");
var DSP_NETCUP = NewDnsProvider("netcup");

D("example.com", REG_NONE, DnsProvider(DSP_NETCUP),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Caveats
Netcup does not allow any TTLs to be set for individual records. Thus in
the diff/preview it will always show a TTL of 0. `NS` records are also
not currently supported.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - get-zones: ❌
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❔
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❔
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
