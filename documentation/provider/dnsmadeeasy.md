## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `DNSMADEEASY`
along with your `api_key` and `secret_key`. More info about authentication can be found in [DNS Made Easy API docs](https://api-docs.dnsmadeeasy.com/).

Example:

{% code title="creds.json" %}
```json
{
  "dnsmadeeasy": {
    "TYPE": "DNSMADEEASY",
    "api_key": "1c1a3c91-4770-4ce7-96f4-54c0eb0e457a",
    "secret_key": "e2268cde-2ccd-4668-a518-8aa8757a65a0"
  }
}
```
{% endcode %}

## Records

ALIAS/ANAME records are supported.

This provider does not support HTTPRED records.

SPF records are ignored by this provider. Use TXT records instead.

## Metadata
This provider does not recognize any special metadata fields unique to DNS Made Easy.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_DNSMADEEASY = NewDnsProvider("dnsmadeeasy");

D("example.com", REG_NONE, DnsProvider(DSP_DNSMADEEASY),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
You can generate your `api_key` and `secret_key` in [Control Panel](https://cp.dnsmadeeasy.com/) in Account Information in Config menu.

API is only available for Business plan and higher plans.

## Caveats

### Global Traffic Director
Global Traffic Director feature is not supported.

## Development

### Debugging
Set `DNSMADEEASY_DEBUG_HTTP` environment variable to dump all API calls made by this provider.

### Testing
Set `sandbox` key to any non-empty value in credentials JSON alongside `api_key` and `secret_key` to make all API calls against DNS Made Easy sandbox environment.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
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
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
