This is the provider for [AdGuardHome](https://github.com/AdguardTeam/AdGuardHome).

## Important notes

This provider only supports the following record types.

* [A](../language-reference/domain-modifiers/A.md)
* [AAAA](../language-reference/domain-modifiers/AAAA.md)
* [CNAME](../language-reference/domain-modifiers/CNAME.md)
* [ALIAS](../language-reference/domain-modifiers/ALIAS.md)
* [ADGUARDHOME_A_PASSTHROUGH](../language-reference/domain-modifiers/ADGUARDHOME_A_PASSTHROUGH.md)
* [ADGUARDHOME_AAAA_PASSTHROUGH](../language-reference/domain-modifiers/ADGUARDHOME_AAAA_PASSTHROUGH.md)

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `ADGUARDHOME`.

Required fields include:

* `username` and `password`: Authentication information
* `host`: The hostname/address of AdGuard Home instance

Example:

{% code title="creds.json" %}
```json
{
  "adguard_home": {
    "TYPE": "ADGUARDHOME",
    "username": "admin",
    "password": "your-password",
    "host": "https://foo.com"
  }
}
```
{% endcode %}

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_ADGUARDHOME = NewDnsProvider("adguard_home");

D("example.com", REG_NONE, DnsProvider(DSP_ADGUARDHOME),
    A("foo", "1.2.3.4"),
    AAAA("another", "2003::1"),
    ALIAS("@", "www.example.com."),
    CNAME("myalias", "www.example.com."),
    ADGUARDHOME_A_PASSTHROUGH("abc", ""),
    ADGUARDHOME_AAAA_PASSTHROUGH("abc", ""),
);
```
{% endcode %}

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❌
  - get-zones: ❌
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
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
