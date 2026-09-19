## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to
`OPENPROVIDER`, together with the username and password of an OpenProvider
account that has API access.

{% code title="creds.json" %}
```json
{
  "openprovider": {
    "TYPE": "OPENPROVIDER",
    "username": "$OPENPROVIDER_USERNAME",
    "password": "$OPENPROVIDER_PASSWORD"
  }
}
```
{% endcode %}

The provider authenticates automatically through OpenProvider's REST login
endpoint. It caches the returned bearer token for the DNSControl process and
authenticates again if OpenProvider rejects an expired token. A bearer token
must not be added to `creds.json`.

The production API defaults to `https://api.openprovider.eu/v1`. The optional
`api_url` field can select another OpenProvider environment. For example, the
OpenProvider sandbox currently uses:

```json
"api_url": "https://api.sandbox.openprovider.nl/v1beta"
```

## Metadata

This provider does not recognize any provider-specific record metadata.

## Usage

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_OPENPROVIDER = NewDnsProvider("openprovider");

D("example.com", REG_NONE, DnsProvider(DSP_OPENPROVIDER),
    A("@", "192.0.2.1"),
    AAAA("www", "2001:db8::1"),
    CNAME("blog", "www.example.com."),
    MX("@", 10, "mail.example.com."),
    TXT("@", "v=spf1 -all"),
    CAA("@", "issue", "letsencrypt.org"),
    SRV("_sip._tcp", 10, 20, 5060, "sip.example.com."),
);
```
{% endcode %}

## Supported record types

The provider supports `A`, `AAAA`, `CAA`, `CNAME`, `MX`, `SPF`, `SRV`, `TLSA`,
and `TXT` records. OpenProvider represents `SPF` as a distinct record type;
an SPF policy can also be published as a `TXT` record.

CAA records are limited to flag `0` and the OpenProvider-documented `issue`,
`issuewild`, and `iodef` tags.

OpenProvider does not support `ALIAS`/`ANAME`, `DNAME`, `DS`, `DNSKEY`, `HTTPS`,
`LOC`, `NAPTR`, `PTR`, `SSHFP`, or `SVCB` records through its standard DNS zone
API. `NS` records are conditionally available through OpenProvider support, but
DNSControl treats the provider-managed authoritative nameservers as read-only.
Unsupported or non-manageable types are rejected during DNSControl's record
audit before a push reaches the API.

## New domains and get-zones

`dnscontrol get-zones` is supported and uses the paginated OpenProvider zone and
record APIs.

DNSControl can create a missing standard master zone. OpenProvider requires a
new master zone to contain at least one record, so the domain configuration must
not be empty on its first push. Zone creation only provisions DNS hosting; it
does not register, transfer, renew, delegate, or otherwise modify the domain at
the registrar.

## Limitations

### TTLs

OpenProvider's current minimum record TTL is 900 seconds. DNSControl raises a
lower requested TTL to 900 seconds and prints a warning. This normalization is
applied before reconciliation so the next preview is clean.

### SOA and NS records

OpenProvider creates and manages the zone's SOA and authoritative apex NS
records. DNSControl reads the apex NS records for nameserver discovery but does
not attempt to add, change, or remove them. It also does not support NS
delegation records for child labels. The provider omits the managed records
when reading a zone and rejects configured `SOA` or child-label `NS` records
rather than repeatedly planning an update the API cannot perform.

`GetNameservers` reads the provider-managed apex NS records returned for the
zone. Dual hosting is not supported because those records cannot be changed.

### Record updates

The API updates individual records. It does not expose stable record IDs, so
the provider identifies an existing record using the complete record value
returned by OpenProvider and sends that exact value in update and remove
requests.

### DNSSEC

OpenProvider can manage DNSSEC outside DNSControl, but `AUTODNSSEC_ON` and
direct `DNSKEY`/`DS` record management are not implemented by this provider.

### Concurrent operations

Concurrent zone gathering has not been verified. Authentication tokens are
cached per provider instance and protected for concurrent access, but the
provider remains marked as concurrency-unverified until the documented race
test has been performed against several zones.

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
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
