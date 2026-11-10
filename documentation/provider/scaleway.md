## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `SCALEWAY`
along with a Scaleway API key (access key + secret key). You can generate one
in the Scaleway console under **Identity and Access Management → API Keys**.

Example:

{% code title="creds.json" %}
```json
{
  "scaleway": {
    "TYPE": "SCALEWAY",
    "access_key": "SCWXXXXXXXXXXXXXXXXX",
    "secret_key": "11111111-2222-3333-4444-555555555555"
  }
}
```
{% endcode %}

The `project_id` field is optional. It scopes the client to a specific Scaleway
project; if omitted, the project associated with the API key is used:

{% code title="creds.json" %}
```json
{
  "scaleway": {
    "TYPE": "SCALEWAY",
    "access_key": "$SCALEWAY_ACCESS_KEY",
    "secret_key": "$SCALEWAY_SECRET_KEY",
    "project_id": "$SCALEWAY_PROJECT_ID"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Scaleway.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE("scaleway"),
    A("test", "1.2.3.4"),
    MX("@", 10, "mail.example.com."),
);
```
{% endcode %}

## Activation

DNSControl uses the [Scaleway Domains & DNS API](https://www.scaleway.com/en/developers/api/domains-and-dns/)
to manage records. You need a Scaleway account with a DNS zone already created
in the console — this provider does not create zones.

## New domains

Zones must be created via the Scaleway console or API before they can be
managed by DNSControl.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Record types
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`CSYNC`](../language-reference/domain-modifiers/CSYNC.md): ❌
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`OPENPGPKEY`](../language-reference/domain-modifiers/OPENPGPKEY.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`RP`](../language-reference/domain-modifiers/RP.md): ❌
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
<!-- provider-features-end -->
