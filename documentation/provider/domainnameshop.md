## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `DOMAINNAMESHOP`
along with your [Domainnameshop Token and Secret](https://www.domeneshop.no/admin?view=api).

Example:

{% code title="creds.json" %}
```json
{
  "mydomainnameshop": {
    "TYPE": "DOMAINNAMESHOP",
    "token": "your-domainnameshop-token",
    "secret": "your-domainnameshop-secret"
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to Domainnameshop.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_DOMAINNAMESHOP = NewDnsProvider("mydomainnameshop");

D("example.com", REG_NONE, DnsProvider(DSP_DOMAINNAMESHOP),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
[Create API Token and secret](https://www.domeneshop.no/admin?view=api)

## Limitations

- Domainnameshop DNS only supports TTLs which are a multiple of 60.
## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❔
  - [get-zones](../commands/get-zones.md): ❔
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
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
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❔
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
