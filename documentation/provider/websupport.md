# Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `WEBSUPPORT`
along with your WebSupport API key and secret. Both are generated in the
[Security section](https://admin.websupport.sk/en/auth/security) of the WebSupport
admin console.

Example:

{% code title="creds.json" %}
```json
{
  "websupport": {
    "TYPE": "WEBSUPPORT",
    "api_key": "your-api-key",
    "secret": "your-api-secret"
  }
}
```
{% endcode %}

You can also use environment variables:

{% code title="creds.json" %}
```json
{
  "websupport": {
    "TYPE": "WEBSUPPORT",
    "api_key": "$WEBSUPPORT_API_KEY",
    "secret": "$WEBSUPPORT_SECRET"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to WebSupport.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_WEBSUPPORT = NewDnsProvider("websupport");

D("example.com", REG_NONE, DnsProvider(DSP_WEBSUPPORT),
    A("@", "1.2.3.4"),
    CNAME("www", "@"),
    MX("@", 10, "mail.example.com."),
);
```
{% endcode %}

# Activation

DNSControl uses the [WebSupport REST API v2](https://rest.websupport.sk/v2/docs)
to manage your DNS records. Generate an API key and secret in the
[Security section](https://admin.websupport.sk/en/auth/security) of the admin
console.

Authentication uses HTTP Basic auth where the username is the API key and the
password is a per-request HMAC-SHA1 signature derived from the secret. The
secret itself is never transmitted.

## Notes and limitations

* **Zones must already exist.** The API has no endpoint to create a zone or to
  enumerate all zones, so `create-domains` and `get-zones` are not supported.
  Add domains through the WebSupport portal first.
* **Supported record types:** `A`, `AAAA`, `CNAME`, `MX`, `TXT`, and `SRV`.
* **Unsupported record types**, due to WebSupport v2 API limitations:
  * `NS` — the API silently ignores attempts to create NS records, so they are
    rejected to avoid an endless create loop. Apex nameservers are managed by
    WebSupport and are not exposed through the DNS record API.
  * `CAA` — the API does not return the `tag`/`flags` of a CAA record on read,
    so the record cannot be managed without churn.
  * `ALIAS`/`ANAME` — WebSupport only allows `ANAME` at the apex and rejects it
    when other apex records exist, so it cannot be supported generically.
  * `LOC`, `NAPTR`, `PTR`, `SSHFP`, `TLSA`, `DS`.
* The provider automatically resolves each domain to its numeric WebSupport
  service id (used internally by the v2 API); you only need to supply
  `api_key` and `secret`.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ❌
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
