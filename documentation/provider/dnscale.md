## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `DNSCALE`
along with your DNScale API key.

Example:

{% code title="creds.json" %}
```json
{
  "dnscale": {
    "TYPE": "DNSCALE",
    "api_key": "dnscale_your-api-key-here"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to DNScale.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_DNSCALE = NewDnsProvider("dnscale");

D("example.com", REG_NONE, DnsProvider(DSP_DNSCALE),
    A("@", "192.0.2.1"),
    A("www", "192.0.2.1"),
    AAAA("@", "2001:db8::1"),
    CNAME("blog", "example.github.io."),
    MX("@", 10, "mail.example.com."),
    TXT("@", "v=spf1 include:_spf.google.com ~all"),
    CAA("@", "issue", "letsencrypt.org"),
END);
```
{% endcode %}

## Activation

DNScale requires an API key which can be obtained from your [DNScale dashboard](https://app.dnscale.eu/dashboard).

## Supported Record Types

DNScale supports the following record types:

- A
- AAAA
- ALIAS
- CAA
- CNAME
- HTTPS
- MX
- NS
- PTR
- SRV
- SSHFP
- SVCB
- TLSA
- TXT

## Nameservers and apex NS records

DNScale automatically assigns nameservers (e.g. `ns1.dnscale.eu`, `ns2.dnscale.eu`) when a zone is created. These system-managed NS records at the zone apex are invisible to DNSControl — they cannot be modified or deleted.

Third-party NS records at the apex **are** supported for multi-provider DNS setups. For example, if you use DNScale alongside another provider, you can add their nameservers as NS records and DNSControl will manage them normally.

### Multi-provider DNS setup

Because DNScale assigns nameservers server-side, `GetNameservers` returns an empty list. This means DNScale's nameservers are not automatically included in registrar delegation. For multi-provider setups, you must explicitly declare them using `NAMESERVER()`:

{% code title="dnsconfig.js" %}
```javascript
var REG_NAMECHEAP = NewRegistrar("namecheap");
var DSP_DNSCALE = NewDnsProvider("dnscale");
var DSP_CLOUDFLARE = NewDnsProvider("cloudflare");

D("example.com", REG_NAMECHEAP,
    DnsProvider(DSP_DNSCALE),
    DnsProvider(DSP_CLOUDFLARE),
    NAMESERVER("ns1.dnscale.eu"),
    NAMESERVER("ns2.dnscale.eu"),
    A("@", "192.0.2.1"),
    A("www", "192.0.2.1"),
END);
```
{% endcode %}

## New domains

If a domain does not exist in your DNScale account, DNSControl will automatically create it when you run `dnscontrol push`.

## API Documentation

For more information about the DNScale API, see the [DNScale API documentation](https://dnscale.eu/api/overview).

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❌
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
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
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
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
