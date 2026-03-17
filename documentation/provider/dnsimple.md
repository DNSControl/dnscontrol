## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `DNSIMPLE`
along with a DNSimple account access token.

You can also set the `baseurl` to use [DNSimple's free sandbox](https://developer.dnsimple.com/sandbox/) for testing.

Examples:

{% code title="creds.json" %}
```json
{
  "dnsimple": {
    "TYPE": "DNSIMPLE",
    "token": "your-dnsimple-account-access-token"
  },
  "dnsimple_sandbox": {
    "TYPE": "DNSIMPLE",
    "baseurl": "https://api.sandbox.dnsimple.com",
    "token": "your-sandbox-account-access-token"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to DNSimple.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_DNSIMPLE = NewRegistrar("dnsimple");
var DSP_DNSIMPLE = NewDnsProvider("dnsimple");

D("example.com", REG_DNSIMPLE, DnsProvider(DSP_DNSIMPLE),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation

DNSControl depends on a DNSimple account access token.

## Caveats

### TXT record length

The DNSimple API supports TXT records of up to 1000 "characters" (assumed to
be octets, per DNS norms, not Unicode characters in an encoding).

See https://support.dnsimple.com/articles/txt-record/

## Development

### Debugging

Set `DNSIMPLE_DEBUG_HTTP` environment variable to `1` to dump all API calls made by this provider.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
