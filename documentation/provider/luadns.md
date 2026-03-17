## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `LUADNS`
along with your [email and API key](https://www.luadns.com/api.html#authentication).

Example:

{% code title="creds.json" %}
```json
{
  "luadns": {
    "TYPE": "LUADNS",
    "email": "your-email",
    "apikey": "your-api-key"
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to LuaDNS.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_LUADNS = NewDnsProvider("luadns");

D("example.com", REG_NONE, DnsProvider(DSP_LUADNS),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
[Create API key](https://api.luadns.com/api_keys).

## Caveats
- LuaDNS cannot change the default nameserver TTL in `nameserver_ttl`, it is forced to fixed at 86400("1d").
This is not the case if you are using vanity nameservers.
- This provider does not currently support the "FORWARD" and "REDIRECT" record types.
- The API is available on the LuaDNS free plan, but due to the limit of 30 records, some tests will fail when doing integration tests.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
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
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
