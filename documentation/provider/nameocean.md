This is the provider for [NameOcean](https://nameocean.net/) DNS hosting. It uses the NameOcean API v1.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `NAMEOCEAN` along with a NameOcean API key.

Example:

{% code title="creds.json" %}
```json
{
  "nameocean": {
    "TYPE": "NAMEOCEAN",
    "token": "$NAMEOCEAN_API_KEY"
  }
}
```
{% endcode %}

| Field | Required | Description |
| --- | --- | --- |
| `token` | yes | A NameOcean API key. |
| `apiurl` | no | The API base URL. Defaults to `https://nameocean.net/api/v1`. |

Create the API key at [nameocean.net/dashboard/api](https://nameocean.net/dashboard/api/). The key needs the `dns:read` and `dns:write` scopes. List the domains DNSControl manages under "Only these domains" so the key cannot touch anything else in the account. NameOcean only accepts API requests from accounts that have two-step login turned on; without it, every request fails with HTTP 401 or 403.

## Metadata

This provider does not recognize any special metadata fields unique to NameOcean.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_NAMEOCEAN = NewDnsProvider("nameocean");

D("example.com", REG_NONE, DnsProvider(DSP_NAMEOCEAN),
    A("@", "192.0.2.1"),
    CNAME("www", "@"),
    MX("@", 10, "mail.example.com."),
);
```
{% endcode %}

`dnscontrol create-domains` (and `push` when the zone is missing) creates the zone in the NameOcean account. `dnscontrol get-zones` lists the zones in the account.

## How changes are applied

The provider downloads the whole zone and uploads the whole desired zone in one request. NameOcean applies that request in a single transaction, so either every change lands or none does.

The upload carries the zone serial from the download. If the zone changed in between (for example in the NameOcean dashboard or from another `dnscontrol push`), NameOcean refuses the upload with `zone_changed` and nothing is modified. Run `dnscontrol preview`/`push` again.

## TTLs

NameOcean accepts a fixed set of TTLs: 300 (5m), 900 (15m), 1800 (30m), 3600 (1h) and 86400 (1d) seconds. TXT records may also use 30 seconds. Any other TTL is rounded to the nearest of these (a TTL exactly in between goes to the shorter one), and DNSControl prints a warning when it does that. Set TTLs to one of these values to avoid the warning.

## Records the provider leaves alone

NameOcean also has HTTP redirects (`REDIRECT`), which are a NameOcean feature rather than a DNS record type, and PTR records keyed by IP address. DNSControl does not manage either: they are not shown in `preview` and every push sends them back unchanged, so a push never deletes them. Manage them in the NameOcean dashboard or API.

## Limitations

* The apex NS records are always `ns1.nameocean.net` through `ns4.nameocean.net` and cannot be changed, so dual hosting is not supported. Apex `NS()` records in `dnsconfig.js` are ignored with a warning; NS records for delegated subdomains work.
* `DS` records are supported for delegated subdomains only. For the zone's own DNSSEC, use the NameOcean dashboard (`AUTODNSSEC_ON` is not implemented yet).
* TXT values cannot contain double quotes, start or end with a space, or be longer than 506 characters.
* CAA records accept the `issue`, `issuewild` and `iodef` tags, with flags `0` or `128`.
* A label can hold only one SRV record.
* NameOcean manages some records itself (for example the address behind an HTTP redirect and ACME challenge TXT records). They are not part of the zone document and DNSControl does not see or change them.

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TXT](../language-reference/domain-modifiers/TXT.md)
    - Not supported: [PTR](../language-reference/domain-modifiers/PTR.md) (NameOcean keys PTR records by IP address, not by DNS label)
<!-- provider-features-end -->
