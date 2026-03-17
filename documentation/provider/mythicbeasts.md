This is the provider for [Mythic Beasts](https://www.mythic-beasts.com/) using its [Primary DNS API v2](https://www.mythic-beasts.com/support/api/dnsv2).

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `MYTHICBEASTS` along with a Mythic Beasts API key ID and secret.

Example:

{% code title="creds.json" %}
```json
{
  "mythicbeasts": {
    "TYPE": "MYTHICBEASTS",
	"keyID": "xxxxxxx",
	"secret": "xxxxxx"
  }
}
```
{% endcode %}

## Usage

For each domain:

* Domains must be added in the [web UI](https://www.mythic-beasts.com/customer/domains), and have DNS enabled.
* In Mythic Beasts' DNS management web UI, new domains will have set a default DNS template of "Mythic Beasts nameservers only". You must set this to "None".

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_MYTHIC = NewDnsProvider("mythicbeasts");

D("example.com", REG_NONE, DnsProvider(DSP_MYTHIC),
    A("test", "1.2.3.4"),
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
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ❌
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
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
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
