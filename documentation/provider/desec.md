## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `DESEC` along with a deSEC account auth token.

Example:

{% code title="creds.json" %}
```json
{
  "desec": {
    "TYPE": "DESEC",
    "auth-token": "your-deSEC-auth-token"
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to deSEC.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_DESEC = NewDnsProvider("desec");

D("example.com", REG_NONE, DnsProvider(DSP_DESEC),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
DNSControl depends on a deSEC account auth token. This token can be obtained by [logging in via the deSEC API](https://desec.readthedocs.io/en/latest/auth/account.html#log-in).

{% hint style="warning" %}
deSEC enforces a daily limit of 300 RRset creation/deletion/modification per
domain. Large changes may have to be done over the course of a few days.  The
integration test suite can not be run in a single session. See
[https://desec.readthedocs.io/en/latest/rate-limits.html#api-request-throttling](https://desec.readthedocs.io/en/latest/rate-limits.html#api-request-throttling)
{% endhint %}

Upon domain creation, the DNSKEY and DS records needed for DNSSEC setup are printed in the command output. If you need these values later, get them from the deSEC web interface or query deSEC nameservers for the CDS records. For example: `dig +short @ns1.desec.io example.com CDS` will return the published CDS records which can be used to insert the required DS records into the parent zone.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ✅
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ✅
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ✅
  - [`DS`](../language-reference/domain-modifiers/DS.md): ✅
<!-- provider-features-end -->
