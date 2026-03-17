This is the provider for [UniFi Network](https://ui.com/), Ubiquiti's network management platform.

UniFi Network includes a local DNS server that can be managed via its API. This provider allows DNSControl to manage DNS records on your UniFi Network controller.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `UNIFI` along with your connection details.

### Configuration parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `TYPE` | Yes | Must be set to `UNIFI` |
| `api_key` | Yes | UniFi API key |
| `host` | Yes* | Controller address for local access (e.g., `https://192.168.1.1`) |
| `console_id` | Yes* | Console ID for cloud access via ui.com |
| `site` | No | Site name (defaults to `default`) |
| `api_version` | No | API version: `auto`, `new`, or `legacy` (defaults to `auto`) |
| `skip_tls_verify` | No | Set to `true` to skip TLS certificate verification |
| `debug` | No | Set to `true` to enable debug output |

*Either `host` or `console_id` is required, but not both.

### Local access example

Use `host` to connect directly to your UniFi controller:

{% code title="creds.json" %}
```json
{
  "unifi": {
    "TYPE": "UNIFI",
    "host": "https://192.168.1.1",
    "api_key": "your-api-key",
    "site": "default"
  }
}
```
{% endcode %}

### Cloud access example

Use `console_id` to connect via UniFi Cloud (ui.com):

{% code title="creds.json" %}
```json
{
  "unifi": {
    "TYPE": "UNIFI",
    "console_id": "28704E24-XXXX-XXXX-XXXX-XXXXXXXXXXXX:1234567890",
    "api_key": "your-api-key",
    "site": "default"
  }
}
```
{% endcode %}

The `console_id` can be found in the URL when accessing your console via https://unifi.ui.com.

## API versions

UniFi Network has two different DNS APIs. The provider supports both and can auto-detect which one to use.

### Legacy API

- **Availability**: UniFi Network 8.x and later
- **Endpoint**: `/proxy/network/v2/api/site/{site}/static-dns`
- **Features**: Basic CRUD operations, update requires delete + create
- **Record types**: A, AAAA, CNAME, MX, TXT, SRV, NS

### New API

- **Availability**: UniFi Network 10.1+ (currently in Early Access)
- **Endpoint**: `/proxy/network/integration/v1/sites/{siteId}/dns/policies`
- **Features**: Full CRUD with native update support
- **Record types**: A, AAAA, CNAME, MX, TXT, SRV

### Choosing an API version

The `api_version` parameter controls which API to use:

| Value | Behavior |
|-------|----------|
| `auto` (default) | Auto-detect: tries new API first, falls back to legacy |
| `new` | Force new API (requires UniFi Network 10.1+) |
| `legacy` | Force legacy API (works with UniFi Network 8.x+) |

**Recommendation**: Use `auto` (the default) for maximum compatibility. The provider will automatically use the best available API for your controller version.

{% code title="creds.json" %}
```json
{
  "unifi": {
    "TYPE": "UNIFI",
    "host": "https://192.168.1.1",
    "api_key": "your-api-key",
    "api_version": "auto"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to UniFi.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_UNIFI = NewDnsProvider("unifi");

D("example.lan", REG_NONE, DnsProvider(DSP_UNIFI),
    A("server", "192.168.1.10"),
    AAAA("server", "fd00::10"),
    CNAME("www", "server.example.lan."),
    MX("@", 10, "mail.example.lan."),
    TXT("@", "v=spf1 mx -all"),
    SRV("_http._tcp", 0, 0, 80, "server.example.lan."),
);
```
{% endcode %}

## Activation

To create an API key for DNSControl:

1. Log in to your UniFi controller
2. Navigate to **Settings** > **Admins & Users**
3. Click on your user profile or create a dedicated API user
4. Generate an API key with appropriate permissions
5. Copy the API key to your `creds.json`

For cloud access, you can also generate API keys at https://unifi.ui.com under your account settings.

## Supported record types

| Type | Legacy API | New API |
|------|------------|---------|
| A | Yes | Yes |
| AAAA | Yes | Yes |
| CNAME | Yes | Yes |
| MX | Yes | Yes |
| TXT | Yes | Yes |
| SRV | Yes | Yes |
| NS | Yes | No |

## Limitations

### No zone concept

UniFi Network stores DNS records flat, without the concept of zones. DNSControl filters records by domain suffix to simulate zone management. This means:

- `dnscontrol get-zones` is not supported
- Creating new domains is not supported (records are added directly)

### Wildcard CNAMEs

UniFi does not support wildcard CNAME records. Attempting to create a `*.example.com` CNAME will result in an error.

### TTL handling

- If TTL is not specified, the provider uses a default of 300 seconds
- TTL support varies by record type in the legacy API (MX and TXT records may ignore TTL)

### Self-signed certificates

If your UniFi controller uses a self-signed certificate, set `skip_tls_verify` to `true`:

{% code title="creds.json" %}
```json
{
  "unifi": {
    "TYPE": "UNIFI",
    "host": "https://192.168.1.1",
    "api_key": "your-api-key",
    "skip_tls_verify": "true"
  }
}
```
{% endcode %}

### Concurrent operations

The provider does not support concurrent API operations. Changes are applied sequentially to ensure reliability.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❌
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❌
  - get-zones: ❌
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❔
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
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
