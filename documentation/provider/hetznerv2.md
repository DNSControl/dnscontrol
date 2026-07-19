## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `HETZNER_V2` along with a [Hetzner API Token](https://docs.hetzner.cloud/reference/cloud#getting-started).

Example:

{% code title="creds.json" %}
```json
{
  "hetzner_v2": {
    "TYPE": "HETZNER_V2",
    "api_token": "your-api-token"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Hetzner DNS API.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_HETZNER = NewDnsProvider("hetzner_v2");

D("example.com", REG_NONE, DnsProvider(DSP_HETZNER),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation

Create a new API Key in the [Hetzner Console](https://docs.hetzner.cloud/reference/cloud#getting-started).

## Caveats

### NS

Removing the Hetzner provided NS records at the root is not possible.

### SOA

Hetzner DNS API does not allow changing the SOA record via their API. There is an alternative method using an import of a full BIND file, but this approach does not play nice with incremental changes or ignored records. At this time you cannot update SOA records via DNSControl.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ✅
<!-- provider-features-end -->
