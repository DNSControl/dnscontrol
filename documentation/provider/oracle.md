## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `ORACLE`
along with other authentication parameters.

Create an API key through the Oracle Cloud portal, and provide the user OCID, tenancy OCID, key fingerprint, region, and the contents of the private key.
The OCID of the compartment DNS resources should be put in can also optionally be provided.

Example:

{% code title="creds.json" %}
```json
{
  "oracle": {
    "TYPE": "ORACLE",
    "compartment": "$ORACLE_COMPARTMENT",
    "fingerprint": "$ORACLE_FINGERPRINT",
    "private_key": "$ORACLE_PRIVATE_KEY",
    "region": "$ORACLE_REGION",
    "tenancy_ocid": "$ORACLE_TENANCY_OCID",
    "user_ocid": "$ORACLE_USER_OCID"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Oracle Cloud.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_ORACLE = NewDnsProvider("oracle");

D("example.com", REG_NONE, DnsProvider(DSP_ORACLE),
    NAMESERVER_TTL(86400),

    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Notes for developers

Integration does not have the capability to set the TTL set differently when Oracle is the provider being tested.
You will see an error message behind displayed, such as below, but it can be safely ignored.

```Text
=== RUN   TestDNSProviders/example.co.uk/Clean_Slate:Empty
WARNING: Oracle Cloud forces TTL=86400 for NS records. Ignoring configured TTL of 300 for ns1.p201.dns.oraclecloud.net.
```

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
