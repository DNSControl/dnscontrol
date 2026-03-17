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

- 0 (Default, currently equivalent to 1209600, or 14 days)
- 300
- 3600
- 7200
- 14400
- 28800
- 57600
- 86400
- 172800
- 345600
- 604800
- 1209600
- 2419200

The provider will automatically round up your TTL to one of these values. For example, 600 seconds would become 3600
seconds, but 300 seconds would stay 300 seconds.

Linode requires [`SRV`](../language-reference/domain-modifiers/SRV.md) records to have a non-zero priority.

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
  - get-zones: ✅
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
