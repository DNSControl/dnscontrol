## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `CONSTELLIX`
along with your Constellix API key and secret key.

Example:

{% code title="creds.json" %}
```json
{
  "constellix": {
    "TYPE": "CONSTELLIX",
    "api_key": "your-constellix-api-key",
    "secret_key": "your-constellix-secret-key"
  }
}
```
{% endcode %}

You can also use environment variables:

```shell
export CONSTELLIX_API_KEY=XXXXXXXXX
export CONSTELLIX_SECRET_KEY=YYYYYYYYY
```

{% code title="creds.json" %}
```json
{
  "constellix": {
    "TYPE": "CONSTELLIX",
    "api_key": "$CONSTELLIX_API_KEY",
    "secret_key": "$CONSTELLIX_SECRET_KEY"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Constellix.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_CONSTELLIX = NewDnsProvider("constellix");

D("example.com", REG_NONE, DnsProvider(DSP_CONSTELLIX),
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

DNSControl talks to the [Constellix DNS v4 API](https://api.dns.constellix.com/v4/docs),
which needs an API key and a secret key.

1. Sign in to the [Constellix management portal](https://dns.constellix.com/).
2. Open your user profile (**Manage Profile**, or **Edit Profile** under your
   username) and choose **Generate API Keys**.
3. Copy both keys into `creds.json` as shown above. The secret key is only
   shown to the user who generated it, so save it when it is displayed.

Every request is signed with a token that embeds the current time. If the API
rejects the keys (HTTP 401), check the keys and make sure the system clock is
correct.

## Supported record types

| Name  | Description |
| ----- | ----------- |
| A     | IPv4 address record |
| AAAA  | IPv6 address record |
| ALIAS | Constellix `ANAME` record |
| CAA   | Certification Authority Authorization record (tags `issue`, `issuewild` and `iodef`) |
| CNAME | Canonical name (alias) record |
| MX    | Mail exchange record |
| NS    | Name server record |
| PTR   | Pointer record, in reverse zones only |
| SRV   | Service record |
| TXT   | Text record |

No other record type is supported.

## New domains

If a domain does not exist in your Constellix account, DNSControl will add it
with the `push` command (or `create-domains`), using Constellix's defaults. The
new zone is read back once for its nameservers. If that read fails, the run
stops with an error saying that the zone was created but is not readable yet;
run `push` again.

## Limitations

### Rate limit

The Constellix API allows a window of 180 units per 60 seconds, shared by the
whole account: every client using any API key of the account draws from the
same window. One request can consume several units, so the effective request
rate can be well below 180 requests per minute.

When the API reports that the window is used up (HTTP 429, or a response that
says no units remain), DNSControl pauses all of its requests until the window
resets, as the response headers say, and then sends the rejected request
again. It prints a notice such as `CONSTELLIX: rate limit reached, waiting 43s`
while it waits. A single request gives up after waiting five minutes in total.

Each request takes one to two seconds, so a single zone never comes near the
limit. Runs over many zones, or on an account where other automation uses the
API at the same time, can be slow. For accounts with many zones, run
DNSControl with a small `--cmax`, such as 3 to 5, so that fewer requests
compete for the window.

### Records DNSControl does not manage

Constellix records can use features that DNSControl does not model. Rather
than overwrite them on the next change, reading a zone fails with an error that
names the record when it finds one of these:

* a record in a mode other than standard: failover, pools or round-robin
  failover
* a record in a Global Traffic Director (GTD) region other than the default
  one
* an IP filter
* GeoProximity
* a disabled record, or a record with a disabled value
* a record without any values
* a record type DNSControl does not manage with this provider, such as `HTTP`
  redirects, `NAPTR`, `RP`, `HINFO`, `CERT` or `SPF`

A zone that is linked to a template is refused as a whole.

Convert the record to a standard record, or remove it, in the Constellix
portal (or unlink the template), then run DNSControl again.

Because of this, `dnscontrol get-zones ... all` with a format that reads
records (`js`, `djs`, `zone`, `tsv`) stops at the first zone that uses one of
these features. `--format=nameonly` does not read records and lists every zone.
On such an account, bootstrap `dnsconfig.js` zone by zone.

### Apex NS records

Constellix serves its own nameservers at the zone apex without listing them as
records, and reports them as the zone's nameservers.

* Apex `NS` records that point at the zone's own Constellix nameservers are
  dropped from the configuration, since Constellix serves them anyway.
* Any other apex `NS` record is managed like an ordinary record, which allows
  [dual hosting](../advanced-features/dual-host.md) with a second DNS
  provider. Constellix serves such records in addition to its own
  nameservers, always with a TTL of 86400, whatever TTL is configured; the
  configured TTL is what DNSControl compares, so this causes no change on the
  next run.
* If the zone stores an apex `NS` record that repeats one of Constellix's own
  nameservers (created in the portal, for example), DNSControl removes that
  value once. The answers do not change, because Constellix serves its own
  nameservers anyway.

### NS records pointing at the zone apex

Constellix cannot store an `NS` record whose target is the zone itself, such as
`NS("foo", "example.com.")` in `example.com`: the API answers with an internal
server error. DNSControl reports such a record as an error before contacting
the API. Point the record at another name of the zone (with an `A` or `AAAA`
record) or at a name outside the zone instead.

### ALIAS

`ALIAS` records are written as Constellix `ANAME` records with `skipLookup`
set, so Constellix accepts a target that does not resolve yet, such as one
created in the same `push`. A target that never resolves (a typo, for
example) is accepted too, and the record then answers nothing.

Constellix refuses an `ALIAS` at the apex next to an `A` record at the apex,
and the `push` fails with the API's error.

### PTR

`PTR` records are supported in reverse zones (`in-addr.arpa`, `ip6.arpa`)
only. In a forward zone Constellix refuses them, and the `push` fails with
the API's error `The name is not valid`.

### TXT

TXT values are written in presentation format. Constellix splits a value
longer than 255 characters into several strings itself, sometimes at another
place than DNSControl would; DNSControl compares the joined text, so this
causes no change on the next run.

### TTL

Constellix accepts TTLs up to 2147483647. A larger TTL is lowered to 2147483647.

### DNSSEC

`AUTODNSSEC_ON` is not supported: the Constellix v4 API has no DNSSEC
endpoints.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`CSYNC`](../language-reference/domain-modifiers/CSYNC.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❌
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`RP`](../language-reference/domain-modifiers/RP.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❌
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❌
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❌
  - [`OPENPGPKEY`](../language-reference/domain-modifiers/OPENPGPKEY.md): ❌
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❌
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
