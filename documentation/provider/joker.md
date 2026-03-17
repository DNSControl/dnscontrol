# Joker DNS Provider

## Configuration

To use this provider, add an entry to `creds.json` with your Joker.com credentials:

{% code title="creds.json" %}
```json
{
  "joker": {
    "TYPE": "JOKER",
    "username": "your-username@joker.com",
    "password": "your-password"
  }
}
```
{% endcode %}

You must have a reseller account in joker.com to use the DMAPI.

Alternatively, you can use an API key (if you have created one on the Joker.com website):

{% code title="creds.json" %}
```json
{
  "joker": {
    "TYPE": "JOKER",
    "api-key": "your-api-key"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Joker.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_JOKER = NewDnsProvider("joker");

D("example.tld", REG_NONE, DnsProvider(DSP_JOKER),
    A("test", "1.2.3.4"),
    CNAME("www", "test"),
    MX("@", 10, "mail.example.tld."),
    TXT("_dmarc", "v=DMARC1; p=reject; rua=mailto:dmarc@example.tld"),
END);
```
{% endcode %}

## Limitations

- This provider updates entire zones, not individual records
- Concurrent operations are not supported due to session-based authentication
- Some record types are not supported (see provider capabilities)
- Minimum TTL is 300 seconds

## Notes

- The provider uses Joker's DMAPI (Domain Management API)
- Authentication uses session-based tokens that expire after inactivity
- Zone updates replace the entire zone content
- The provider supports both username/password and API key authentication

## Supported Record Types

- A
- AAAA
- CNAME
- MX
- TXT
- SRV
- CAA
- NAPTR

## Unsupported Record Types

- ALIAS
- DS
- DNSKEY
- HTTPS
- LOC
- PTR
- SOA
- SSHFP
- SVCB
- TLSA

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❌
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ✅
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❌
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❌
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
