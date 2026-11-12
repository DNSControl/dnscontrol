## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `SPACESHIP`
along with an API key and secret created in [Spaceship API Manager](https://www.spaceship.com/application/api-manager/).

Grant the key `dnsrecords:read`, `dnsrecords:write`, `domains:read`, and `domains:write` scopes.

Example:

{% code title="creds.json" %}
```json
{
  "spaceship": {
    "TYPE": "SPACESHIP",
    "api_key": "your-spaceship-api-key",
    "api_secret": "your-spaceship-api-secret"
  }
}
```
{% endcode %}

The [creds.json](../commands/creds-json.md#example-commands) page in the docs explains how you can generate this dynamically so you can pull the secret token from 1Password or the vault of your choosing.

## Metadata

This provider does not recognize any special metadata fields unique to Spaceship.

## Usage

### As DNS Provider only

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE("spaceship"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

### As both Registrar and DNS Provider

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("spaceship"), SERVICE("spaceship"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

### As Registrar only (DNS hosted elsewhere)

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("spaceship"), SERVICE("cloudflare"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

When used as a registrar, Spaceship updates nameserver delegation. If the desired nameservers are Spaceship's hosted set (`launch1.spaceship.net` and `launch2.spaceship.net`), the provider sends `provider: "basic"`. Any other list is sent as `provider: "custom"`.

## Activation

1. Log in to [Spaceship](https://www.spaceship.com/)
2. Open [API Manager](https://www.spaceship.com/application/api-manager/) and create a key
3. Enable the DNS and domain scopes listed above
4. Put the key and secret in `creds.json`

The domain must already exist in the Spaceship account. This provider cannot create zones.

## Rate limiting

Spaceship returns HTTP 429 with a `Retry-After` header when an endpoint's quota is exceeded. Some quotas are tight: domain info is 5 requests per domain per 300 seconds; DNS record reads and writes are 300 per 300 seconds. The provider waits and retries, honoring `Retry-After`, for up to 10 minutes per request.

A large `push` or an integration-test run can therefore pause for minutes. Increase the Go test timeout when running the suite:

```shell
cd integrationTest
go test -timeout 0 -v -args -verbose -profile SPACESHIP
```

## Record types

Writable custom DNS types: A, AAAA, ALIAS, CAA, CNAME, HTTPS, MX, NS (non-apex), PTR, SRV, SVCB, TLSA, TXT.

TTL values are clamped to 60–3600 seconds (Spaceship's API range). A TTL of 0 becomes 3600.

Apex CNAME is allowed. Apex ALIAS is rejected because Spaceship stores it as a CNAME; declare an apex CNAME instead.

Null MX ([RFC 7505](https://www.rfc-editor.org/rfc/rfc7505.html), `MX("@", 0, ".")`) and SRV records whose target is `.` are not supported. The API requires a hostname for those fields and returns HTTP 422 otherwise.

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
