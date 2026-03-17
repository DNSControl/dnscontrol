This is a read-only/monitoring "registrar". It does a DNS NS lookup to confirm the nameserver servers are correct. This "registrar" is unable to update/correct the NS servers but will alert you if they are incorrect. A common use of this provider is when the domain is with a registrar that does not have an API.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `DNSOVERHTTPS`.

{% code title="creds.json" %}
```json
{
  "dohdefault": {
    "TYPE": "DNSOVERHTTPS"
  }
}
```
{% endcode %}

The DNS-over-HTTPS provider defaults to using Google Public DNS however you may configure an alternative RFC 8484 DoH provider using the `host` parameter.

Example:

{% code title="creds.json" %}
```json
{
  "dohcloudflare": {
    "TYPE": "DNSOVERHTTPS",
    "host": "cloudflare-dns.com"
  }
}
```
{% endcode %}

Some common DoH providers are:

* `cloudflare-dns.com` ([Cloudflare](https://developers.cloudflare.com/1.1.1.1/dns-over-https))
* `9.9.9.9` ([Quad9](https://www.quad9.net/about/))
* `dns.google` ([Google Public DNS](https://developers.google.com/speed/public-dns/docs/doh))

## Metadata
This provider does not recognize any special metadata fields unique to DOH.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_MONITOR = NewRegistrar("dohcloudflare");

D("example.com", REG_MONITOR,
  NAMESERVER("ns1.example.com."),
  NAMESERVER("ns2.example.com."),
);
```
{% endcode %}

{% hint style="info" %}
**NOTE**: This checks the NS records via a DNS query.  It does not check the
registrar's delegation (i.e. the `Name Server:` field in whois). In theory
these are the same thing but there may be situations where they are not.
{% endhint %}

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ❌
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❌
  - get-zones: ❔
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❔
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❔
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ❔
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ❔
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❔
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❔
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
