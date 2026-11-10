## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `MISAKA`
along with a [Misaka.io API token](https://app.misaka.io/account/api).

Example:

{% code title="creds.json" %}
```json
{
  "misaka": {
    "TYPE": "MISAKA",
    "api_token": "your-misaka-api-token"
  }
}
```
{% endcode %}

You can also use environment variables:

```shell
export MISAKA_API_TOKEN=XXXXXXXXX
```

{% code title="creds.json" %}
```json
{
  "misaka": {
    "TYPE": "MISAKA",
    "api_token": "$MISAKA_API_TOKEN"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Misaka.io.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_MISAKA = NewDnsProvider("misaka");

D("example.com", REG_NONE, DnsProvider(DSP_MISAKA),
    A("@", "203.0.113.10"),
    A("www", "203.0.113.10"),
    AAAA("@", "2001:db8::1"),
    ALIAS("cdn", "edge.example.net."),
    CNAME("blog", "example.github.io."),
    MX("@", 10, "mail.example.com."),
    TXT("@", "v=spf1 -all"),
    CAA("@", "issue", "letsencrypt.org"),
    SRV("_sip._tcp", 10, 60, 5060, "sip.example.com."),
);
```
{% endcode %}

## Activation

DNSControl talks to the [Misaka.io v3 API](https://docs.misaka.io/api), which
needs an API token.

1. Sign in to the [Misaka.io console](https://app.misaka.io/) and open the
   **API** page of your account.
2. Create a token. A token belongs to one team, and DNSControl manages the zones
   of that team.
3. Put it in `creds.json` as shown above.

The team must have the DNS service enabled. Otherwise every request is answered
with `403 DNS service is not enabled for this Team or your role lacks access.`

{% hint style="warning" %}
Misaka.io labels the v3 API as alpha, still under active development. A
breaking change on their side may need a provider update.
{% endhint %}

## Supported record types

| Name  | Description |
| ----- | ----------- |
| A     | IPv4 address record |
| AAAA  | IPv6 address record |
| ALIAS | Alias record, served as the addresses of its target |
| CAA   | Certification Authority Authorization record |
| CNAME | Canonical name (alias) record |
| DNAME | Delegation name record |
| MX    | Mail exchange record |
| NS    | Name server record, for a delegated child only |
| PTR   | Pointer record |
| SRV   | Service record |
| SSHFP | SSH fingerprint record |
| TLSA  | TLSA record |
| TXT   | Text record |

No other record type is supported.

## New domains

If a domain does not exist in your Misaka.io account, DNSControl will add it
with the `push` command. Misaka.io only accepts names under a public suffix.

## Limitations

### TTLs belong to the record set

A TTL is a property of the record set, not of a single value in it, so every
value under one label and type shares one TTL. This is what DNSControl already
requires of a configuration, so it only matters when reading a zone that was
edited elsewhere.

### Zone apex

The SOA record and the NS records at the zone apex are managed by Misaka.io, so
DNSControl leaves both alone:

* They are not reported by `dnscontrol get-zones`.
* An apex `NS` record that matches one of the zone's nameservers is dropped
  silently. Any other apex `NS` record is dropped with a warning. Dual hosting a
  zone with a second DNS provider is therefore not possible.

The provider reports the nameservers from the zone's apex `NS` records, so no
explicit `NAMESERVER()` is needed for DNSControl to tell the registrar where to
delegate.

### Record sets DNSControl does not manage

Misaka.io record sets can carry features that DNSControl does not model. Rather
than overwrite them on the next change, reading a zone fails with an error that
names the record set when it finds one of these:

* GeoDNS `filters`
* an `alias` (CNAME flattening of a record set)
* a disabled record set (`enabled: false`)
* a protected record set
* a record set without any values
* a hostname value of `@` (Misaka.io publishes it as the literal name `\@.`;
  use the zone's full name instead)
* a record set whose name ends with a dot

Change or remove the record set in the Misaka.io console, then run DNSControl
again.

### Record set names

Misaka.io stores record set names exactly as they were entered. DNSControl
reports a name such as `Www` in lower case, as `www`, and changes or deletes the
record set under the name Misaka.io knows it by.

### TTL

Misaka.io accepts TTLs up to 2147483647. A larger TTL is lowered to 2147483647.

### DNSSEC

`AUTODNSSEC_ON` is not implemented. DNSSEC is switched on per zone in the
Misaka.io console.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
- Record types
  - [`A`](../language-reference/domain-modifiers/A.md) [`AAAA`](../language-reference/domain-modifiers/AAAA.md) [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md) [`CAA`](../language-reference/domain-modifiers/CAA.md) [`CNAME`](../language-reference/domain-modifiers/CNAME.md) [`DNAME`](../language-reference/domain-modifiers/DNAME.md) [`MX`](../language-reference/domain-modifiers/MX.md) [`NS`](../language-reference/domain-modifiers/NS.md) [`PTR`](../language-reference/domain-modifiers/PTR.md) [`SRV`](../language-reference/domain-modifiers/SRV.md) [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md) [`TLSA`](../language-reference/domain-modifiers/TLSA.md) [`TXT`](../language-reference/domain-modifiers/TXT.md)
    - Not supported: [`SOA`](../language-reference/domain-modifiers/SOA.md) (Managed by Misaka.io)
<!-- provider-features-end -->
