This is the provider for the [IONOS hosting DNS API](https://developer.hosting.ionos.com/docs/dns), which serves the DNS of domains hosted at IONOS. It is not the IONOS Cloud DNS API.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `IONOS` along with an IONOS hosting API key.

Example:

{% code title="creds.json" %}
```json
{
  "ionos": {
    "TYPE": "IONOS",
    "api_key": "your-prefix.your-secret"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to IONOS.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_IONOS = NewDnsProvider("ionos");

D("example.com", REG_NONE, DnsProvider(DSP_IONOS),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation

Create an API key on the [IONOS developer portal](https://developer.hosting.ionos.com/keys). The key has the form `<prefix>.<secret>` and can read and change the DNS of every domain of the account.

## Caveats

- The zone must already exist in the IONOS account: this provider does not create zones.
- The lowest accepted TTL is 60 seconds.
- Records that target the root (`MX("@", 0, ".")`, a null `SRV`, a `CNAME` or `NS` to `.`) are rejected.
- `TXT` values that are empty or contain a double quote or a backslash are rejected: the API documentation does not describe how it escapes them.
- IONOS may deactivate records that conflict with its services. A deactivated record that is still declared in `dnsconfig.js` shows up as a change, and `dnscontrol push` enables it again. A deactivated record that is not declared is deleted, like any other undeclared record. `dnscontrol get-zones` exports deactivated records like any other record.
- A record whose content cannot be parsed, for example one entered in the IONOS control panel, is skipped with a warning. `dnscontrol push` neither changes nor deletes it, and `dnscontrol get-zones` does not export it. Declaring the same value in `dnsconfig.js` leaves it alone, with a warning; declaring another value at that name creates a new record, unless one of the two is a `CNAME`: that conflict is an error, reported before anything is written.
- IONOS manages the SOA and the apex NS records. This provider leaves them alone, and a `NAMESERVER()` declaration does not change them.
- The API also lists `CERT`, `DS`, `HTTPS`, `LOC`, `OPENPGPKEY`, `RP`, `SMIMEA`, `SSHFP`, `SVCB`, `TLSA` and `URI`, which this provider does not support yet. Such records cannot be declared in `dnsconfig.js`. They stay in the zone untouched, and `dnscontrol get-zones` does not export them.

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
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`CSYNC`](../language-reference/domain-modifiers/CSYNC.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❌
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`RP`](../language-reference/domain-modifiers/RP.md): ❔
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❌
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`OPENPGPKEY`](../language-reference/domain-modifiers/OPENPGPKEY.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❔
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❔
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
