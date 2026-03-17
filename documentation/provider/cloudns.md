## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `CLOUDNS`
along with your [Api user ID and password](https://www.cloudns.net/wiki/article/42/).

Example:

{% code title="creds.json" %}
```json
{
  "cloudns": {
    "TYPE": "CLOUDNS",
    "auth-id": "12345",
    "sub-auth-id": "12345",
    "auth-password": "your-password"
  }
}
```
{% endcode %}

Current version of provider doesn't support `sub-auth-user`.

## Records

ClouDNS does support DS Record on subdomains (not the apex domain itself).

ClouDNS requires NS records exist for any DS records. No other records for
the same label may exist (A, MX, TXT, etc.). If DNSControl is adding NS and
DS records in the same update, the NS records will be inserted first.

## Metadata
This provider does not recognize any special metadata fields unique to ClouDNS.

## Web Redirects
ClouDNS supports ClouDNS-specific "WR record (web redirects)" for your domains.
Simply use the `CLOUDNS_WR` functions to make redirects like any other record:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_CLOUDNS = NewDnsProvider("cloudns");

D("example.com", REG_NONE, DnsProvider(DSP_CLOUDNS),
  CLOUDNS_WR("@", "http://example.com/"),
  CLOUDNS_WR("www", "http://example.com/"),
);
```
{% endcode %}

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_CLOUDNS = NewDnsProvider("cloudns");

D("example.com", REG_NONE, DnsProvider(DSP_CLOUDNS),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
[Create Auth ID](https://www.cloudns.net/api-settings/).  Only paid account can use API

## Caveats
ClouDNS does not allow all TTLs, only a specific subset of TTLs. By default, the following [TTLs are supported](https://www.cloudns.net/wiki/article/188/):
- 60  (1 minute)
- 300 (5 minutes)
- 900 (15 minutes)
- 1800 (30 minutes)
- 3600 (1 hour)
- 21600 (6 hours)
- 43200 (12 hours)
- 86400 (1 day)
- 172800 (2 days)
- 259200 (3 days)
- 604800 (1 week)
- 1209600 (2 weeks)
- 2419200 (4 weeks)

The provider will automatically round up your TTL to one of these values. For example, 350 seconds would become 900
seconds, but 300 seconds would stay 300 seconds.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ✅
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ✅
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❌
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❌
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❌
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
