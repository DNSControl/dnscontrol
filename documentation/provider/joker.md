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
D("example.tld", REGISTRAR(), SERVICE("joker"),
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

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ❌
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
