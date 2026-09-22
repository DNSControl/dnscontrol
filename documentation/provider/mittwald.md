## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `MITTWALD` along with an [mStudio API token](https://developer.mittwald.de/docs/v2/api/intro/).

Example:

{% code title="creds.json" %}
```json
{
  "mittwald": {
    "TYPE": "MITTWALD",
    "api_token": "your-api-token"
  }
}
```
{% endcode %}

The token belongs to an mStudio user. The provider finds each domain in the projects that user can access.

## Metadata

This provider does not recognize any special metadata fields unique to mittwald.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_MITTWALD = NewDnsProvider("mittwald");

D("example.com", REG_NONE, DnsProvider(DSP_MITTWALD),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation

Create an API token in mStudio under your user profile. A domain can be managed once it is part of an mStudio project; the provider does not create zones.

## Caveats

### One zone per name

mStudio stores every name as its own zone: `example.com`, `www.example.com` and `_dmarc.example.com` are three zones. The provider creates the zone of a name when a record is added to it and deletes it when its last record is removed.

### Record sets

Each name holds one set per type: A and AAAA together, CNAME, MX, TXT, SRV and CAA. All records of a set share one TTL, so A and AAAA records of one name must have the same TTL. A set holds at most 10 A, 10 AAAA, 10 MX or 20 TXT records.

### Records managed by mStudio

mStudio sets the addresses of a name that is connected to an ingress, and the mail exchangers of its mail service. The provider neither shows nor changes these sets. Declaring A, AAAA or MX records for such a name replaces the managed set with the declared records. A CNAME cannot be added to a name that has a managed set.

### Unsupported records

Wildcard names (`*`), NS records, a null MX (`MX("@", 0, ".")`), an MX preference above 100, empty TXT records, TXT records that end with a space, and CAA values other than a host name (no parameters, no `iodef` URL) are not accepted by mStudio and are rejected by `dnscontrol check`.

### TTL

A TTL must be between 60 and 86400 seconds. A set whose TTL is "auto" in mStudio is served with a TTL of 60 and read as such.

### Rate limit

mStudio allows an API user 3000 requests in 600 seconds. When a response says the limit is used up, the provider waits for its reset. A request that is still answered with "too many requests", because another client of the same user used the limit, is repeated for up to 11 minutes.

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
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❌
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
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
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
