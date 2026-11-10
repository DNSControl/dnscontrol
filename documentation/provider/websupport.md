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
D("example.com", REGISTRAR(), SERVICE("websupport"),
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
* **`TXT` records may not end in a space.** The API silently strips trailing
  whitespace, so such records are rejected rather than churning forever.
* **Null MX ([RFC 7505](https://www.rfc-editor.org/rfc/rfc7505.html)) is not
  supported.** A null MX has an empty target and the API rejects a record with
  empty `content`, so `MX("@", 0, ".")` is rejected with a clear error.
* **MX and SRV reads cost one extra API request each.** WebSupport's record
  listing endpoints return `null` for `priority`, `port` and `weight` on every
  record, even though the values are stored correctly and served in DNS. The
  provider re-reads each `MX` and `SRV` record through the v1 single-record
  endpoint, which still reports them. This is a workaround for a bug on
  WebSupport's side and stops by itself once they fix the listing.
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
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Record types
  - [`A`](../language-reference/domain-modifiers/A.md) [`AAAA`](../language-reference/domain-modifiers/AAAA.md) [`CNAME`](../language-reference/domain-modifiers/CNAME.md) [`MX`](../language-reference/domain-modifiers/MX.md) [`SRV`](../language-reference/domain-modifiers/SRV.md) [`TXT`](../language-reference/domain-modifiers/TXT.md)
    - Not supported: [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md) (WebSupport's ANAME is apex-only and conflicts with other apex records.)
    - Not supported: [`CAA`](../language-reference/domain-modifiers/CAA.md) (The v2 API does not return CAA tag/flags on read, so records cannot be managed without churn.)
<!-- provider-features-end -->
