## Configuration

This provider is for [Infoblox NIOS](https://www.infoblox.com/) DDI appliances via the WAPI (Web API). To use this provider, add an entry to `creds.json` with `TYPE` set to `INFOBLOX`.

Example:

{% code title="creds.json" %}
```json
{
  "infoblox": {
    "TYPE": "INFOBLOX",
    "host": "https://grid-master.example.com",
    "username": "dnscontrol",
    "password": "your-password",
    "view": "default",
    "wapi_version": "2.12"
  }
}
```
{% endcode %}

### Parameters

| Field | Required | Default | Description |
| ----- | -------- | ------- | ----------- |
| `host` | Yes | | Base URL of the Infoblox Grid Master (e.g. `https://grid-master.example.com`) |
| `username` | Yes | | WAPI username |
| `password` | Yes | | WAPI password |
| `view` | No | `default` | DNS view to manage |
| `wapi_version` | No | `2.12` | WAPI version string |
| `tls_skip_verify` | No | `false` | Skip TLS certificate verification |
| `ca_cert` | No | | Path to a custom CA certificate file (PEM format) |

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_INFOBLOX = NewDnsProvider("infoblox");

D("example.com", REG_NONE, DnsProvider(DSP_INFOBLOX),
    A("test", "1.2.3.4"),
    AAAA("test6", "2001:db8::1"),
    CNAME("www", "test.example.com."),
    MX("@", 10, "mail.example.com."),
    TXT("@", "v=spf1 -all"),
    SRV("_sip._tcp", 10, 60, 5060, "sip.example.com."),
    CAA("@", "issue", "letsencrypt.org"),
    PTR("4.3.2.1", "host.example.com."),
);
```
{% endcode %}

## Supported record types

The following record types are supported:

- A
- AAAA
- CAA
- CNAME
- MX
- PTR
- SRV
- TXT

## Limitations

- **NS records are not supported.** Infoblox requires an `addresses` field (nameserver IPs) when creating NS delegation records, which DNSControl does not provide.
- **Zones must be pre-created.** DNSControl cannot create zones in Infoblox; they must already exist in the configured DNS view.
- **TXT records are limited to 255 bytes.** Infoblox does not support multi-string TXT records or single strings longer than 255 bytes.
- **Empty TXT records are rejected.**
- **TXT records with backslashes are not supported.**
- **Concurrent operations are not supported.** The provider processes changes sequentially.

## Activation

DNSControl connects to the Infoblox WAPI using HTTP Basic Authentication. Create a local user account on the Infoblox Grid Master with permissions to manage DNS records in the target zone and view.

The minimum required permissions are:

- Read/Write access to the DNS zone(s) you want to manage
- Read access to zone_auth objects (for zone lookup and SOA TTL retrieval)

## New domains

DNSControl cannot create new zones in Infoblox. Zones must already exist in the configured DNS view before DNSControl can manage them.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❌
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
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
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❔
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❔
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
