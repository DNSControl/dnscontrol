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
D("example.com", REGISTRAR(), SERVICE("mythicbeasts"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Record types
  - [`A`](../language-reference/domain-modifiers/A.md) [`AAAA`](../language-reference/domain-modifiers/AAAA.md) [`CAA`](../language-reference/domain-modifiers/CAA.md) [`CNAME`](../language-reference/domain-modifiers/CNAME.md) [`MX`](../language-reference/domain-modifiers/MX.md) [`NS`](../language-reference/domain-modifiers/NS.md) [`PTR`](../language-reference/domain-modifiers/PTR.md) [`SRV`](../language-reference/domain-modifiers/SRV.md) [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md) [`TLSA`](../language-reference/domain-modifiers/TLSA.md) [`TXT`](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
