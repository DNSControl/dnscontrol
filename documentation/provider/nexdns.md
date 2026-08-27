## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `NEXDNS`
along with a [NexDNS API key](https://nexdns.tech/settings/api-keys).

Example:

{% code title="creds.json" %}
```json
{
  "nexdns": {
    "TYPE": "NEXDNS",
    "api_token": "your-nexdns-api-key"
  }
}
```
{% endcode %}

You can also use environment variables:

```shell
export NEXDNS_API_TOKEN=XXXXXXXXX
```

{% code title="creds.json" %}
```json
{
  "nexdns": {
    "TYPE": "NEXDNS",
    "api_token": "$NEXDNS_API_TOKEN"
  }
}
```
{% endcode %}

The base URL of the API can be changed with the optional `api_url` field. It
defaults to `https://api.nexdns.tech/v1`.

## Metadata

This provider does not recognize any special metadata fields unique to NexDNS.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_NEXDNS = NewDnsProvider("nexdns");

D("example.com", REG_NONE, DnsProvider(DSP_NEXDNS),
    A("@", "203.0.113.10"),
    A("www", "203.0.113.10"),
    AAAA("@", "2001:db8::1"),
    CNAME("blog", "example.github.io."),
    MX("@", 10, "mail.example.com."),
    TXT("@", "v=spf1 -all"),
    CAA("@", "issue", "letsencrypt.org"),
    SRV("_sip._tcp", 10, 60, 5060, "sip.example.com."),
);
```
{% endcode %}

## Activation

DNSControl talks to the [NexDNS API](https://nexdns.tech/docs/api), which needs
an API key.

1. Sign in and open **Settings**, then **API keys**.
2. Create a key holding the `zones.read`, `zones.write`, `records.read` and
   `records.write` scopes. The key is shown once, when it is created.
3. Put it in `creds.json` as shown above.

The API is available on a plan that includes API access; see the
[pricing page](https://nexdns.tech/pricing) for which plans those are.

## Supported record types

| Name  | Description |
| ----- | ----------- |
| A     | IPv4 address record |
| AAAA  | IPv6 address record |
| ALIAS | Alias record |
| CAA   | Certification Authority Authorization record |
| CNAME | Canonical name (alias) record |
| DNAME | Delegation name record |
| DS    | Delegation signer record, for a delegated child only |
| MX    | Mail exchange record |
| NS    | Name server record, for a delegated child only |
| PTR   | Pointer record |
| SRV   | Service record |
| TLSA  | TLSA record |
| TXT   | Text record |

No other record type is supported.

## New domains

If a domain does not exist in your NexDNS account, DNSControl will add it with
the `push` command.

## Limitations

### TTLs belong to the record set

A TTL is a property of the record set, not of a single value in it, so every
value under one label and type shares one TTL. This is what DNSControl already
requires of a configuration, so it only matters when reading a zone that was
edited elsewhere.

### Zone apex

The SOA record and the NS records at the zone apex are maintained by NexDNS and
cannot be written through the API, so DNSControl leaves both alone:

* They are not reported by `dnscontrol get-zones`.
* An apex `NS` record that matches one of the zone's declared nameservers is
  dropped silently. Any other apex `NS` record is dropped with a warning,
  because the API would refuse it. Dual hosting a zone with a second DNS
  provider is therefore not possible.

The provider reports the nameservers the zone is served from, so no explicit
`NAMESERVER()` is needed for DNSControl to tell the registrar where to delegate.

### DS records

`DS` records are supported for delegated children. A `DS` record at the zone
apex belongs in the parent zone and is rejected by `dnscontrol check`.

### DNSSEC

`AUTODNSSEC_ON` is not implemented. DNSSEC is switched on per zone outside of
DNSControl.

### Concurrent operations

Zone data is not gathered concurrently. The provider has not been verified for
concurrent use.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ✅
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❌
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❌
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❌
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❌
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
