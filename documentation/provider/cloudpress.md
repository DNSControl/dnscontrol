# Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `CLOUDPRESS`
along with the base URL of your CloudPress instance and a CloudPress
[API key](https://docs.cloudpress.com/API/api-keys/).

CloudPress is a brand/instance-specific platform, so there is no single default
API host. Set `base_url` to the host that serves your account's API (for example
`https://app.cloudpress.com`).

Example:

{% code title="creds.json" %}
```json
{
  "cloudpress": {
    "TYPE": "CLOUDPRESS",
    "base_url": "https://app.cloudpress.com",
    "api_token": "your-cloudpress-api-key"
  }
}
```
{% endcode %}

You can also use environment variables:

```shell
export CLOUDPRESS_BASE_URL=https://app.cloudpress.com
export CLOUDPRESS_API_TOKEN=XXXXXXXXX
```

{% code title="creds.json" %}
```json
{
  "cloudpress": {
    "TYPE": "CLOUDPRESS",
    "base_url": "$CLOUDPRESS_BASE_URL",
    "api_token": "$CLOUDPRESS_API_TOKEN"
  }
}
```
{% endcode %}

## Authentication

The provider authenticates with a CloudPress API key sent as a bearer token
(`Authorization: Bearer <api_token>`). You can create an API key via
`POST /api/api_keys`; the token value is only returned once, so store it
immediately.

OAuth 2.1 access tokens are also accepted by CloudPress, but the OAuth flow is
interactive (PKCE `authorization_code`) and not well suited to an unattended CLI,
so an API key is recommended.

> Trial accounts cannot use the CloudPress API.

## Metadata

Some endpoints (including zone creation) require an account to be selected via
the `X-Auth-Account` header. If you use a user API key and want DNSControl to
create zones, set the optional `account_id` field:

{% code title="creds.json" %}
```json
{
  "cloudpress": {
    "TYPE": "CLOUDPRESS",
    "base_url": "https://app.cloudpress.com",
    "api_token": "$CLOUDPRESS_API_TOKEN",
    "account_id": "12345"
  }
}
```
{% endcode %}

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_CLOUDPRESS = NewDnsProvider("cloudpress");

D("example.com", REG_NONE, DnsProvider(DSP_CLOUDPRESS),
    A("test", "1.2.3.4"),
    MX("@", 10, "mail.example.com."),
);
```
{% endcode %}

# Activation

DNSControl uses the [CloudPress DNS API](https://docs.cloudpress.com/API/dns/) to
manage your DNS records. You will need to generate an API key to use this provider.

## New domains

If a domain does not exist in your CloudPress account, DNSControl will
automatically add it with the `push` command (an `account_id` may be required,
see [Metadata](#metadata)).

## Caveats

- CloudPress manages the apex `NS` records itself; they are exposed through
  `GetNameservers()` and are not managed as ordinary records. Dual-hosting with
  another provider is therefore not supported.
- The provider currently manages the standard record types `A`, `AAAA`, `CNAME`,
  `TXT`, `MX`, `NS`, `SRV`, `CAA` and `PTR`. Other CloudPress-specific types
  (such as Redirect, Flatten, Pull Zone or Script records) are ignored by
  DNSControl and left untouched in the zone.

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
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
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
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
