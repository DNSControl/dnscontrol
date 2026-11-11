## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `HOSTINGDE`
along with your [`authToken` and optionally an `ownerAccountId`](https://www.hosting.de/api/#requests-and-authentication).

Example:

{% code title="creds.json" %}
```json
{
  "hosting.de": {
    "TYPE": "HOSTINGDE",
    "authToken": "YOUR_API_KEY"
  }
}
```
{% endcode %}

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("hosting.de"), SERVICE("hosting.de"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Using this provider with http.net and others

http.net and other DNS service providers use an API that is compatible with hosting.de's API.
Using them requires setting the `baseURL` and (optionally) overriding the default nameservers.

### Example http.net configuration

An example `creds.json` configuration:

{% code title="creds.json" %}
```json
{
  "http.net": {
    "TYPE": "HOSTINGDE",
    "authToken": "YOUR_API_KEY",
    "baseURL": "https://partner.http.net"
  }
}
```
{% endcode %}

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE("http.net", ALL_NS, {
  "default_ns": [
    "ns1.routing.net.",
    "ns2.routing.net.",
    "ns3.routing.net.",
  ],
}));
```
{% endcode %}

#### Why this works

hosting.de has the concept of _nameserver sets_ but this provider does not implement it.
The `HOSTINGDE` provider **ignores the default nameserver set** defined in your account to avoid unintentional changes and consolidate the full configuration in DNSControl.
Instead, it uses hosting.de's nameservers (`ns1.hosting.de.`, `ns2.hosting.de.`, and `ns3.hosting.de.`) by default, regardless of your account settings.
Using the `default_ns` metadata, the default nameserver set can be overwritten.

## Caveats

### SOA

hosting.de derives the primary nameserver of a zone from its nameserver set and offers no way to override it, so the `ns` parameter of [`SOA()`](../language-reference/domain-modifiers/SOA.md) is ignored.

The contact address of a zone lives in the zone configuration rather than in an SOA record, so the mailbox of `SOA()` ends up there.
A mailbox that names its own host is used as the address, so `hostmaster.example.com.` becomes `hostmaster@example.com`.
One without a host is qualified with the zone name.
The address may not end in a period; the API rejects that.

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DS](../language-reference/domain-modifiers/DS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SOA](../language-reference/domain-modifiers/SOA.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
