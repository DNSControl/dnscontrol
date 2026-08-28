## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `LINODE`
along with your [Linode Personal Access Token](https://cloud.linode.com/profile/tokens).

Example:

{% code title="creds.json" %}
```json
{
  "linode": {
    "TYPE": "LINODE",
    "token": "your-linode-personal-access-token"
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to Linode.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_LINODE = NewDnsProvider("linode");

D("example.com", REG_NONE, DnsProvider(DSP_LINODE),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
[Create Personal Access Token](https://cloud.linode.com/profile/tokens)

## Caveats
Linode does not allow all TTLs, but only a specific subset of TTLs. The following TTLs are supported
([source](https://www.linode.com/docs/api/domains/#domains-list__responses)):

- 0 (Default: the _Default TTL_ configured for the zone[^1])
- 30 ("30s")
- 120 ("2m")
- 300 ("5m")
- 3600 ("1h")
- 7200 ("2h")
- 14400 ("4h")
- 28800 ("8h")
- 57600 ("16h")
- 86400 ("1d")
- 172800 ("2d")
- 345600 ("4d")
- 604800 ("1w")
- 1209600 ("2w")
- 2419200 ("4w")

[^1]: The _default TTL_ for a zone may be set using Linode's web UI or API.
This sets the TTL for the SOA, NS records, and any other records that have their TTL set to zero. It also sets the value in the _minimum TTL_ field of the SOA.
There currently seems to be no way to set the minimum TTL a different value than
the _default TTL_.

The provider will automatically round up your TTL to one of these values. For example, 600 seconds would become 3600
seconds, but 300 seconds would stay 300 seconds.

Linode requires [`SRV`](../language-reference/domain-modifiers/SRV.md) records to have a non-zero priority.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❔
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
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❔
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
