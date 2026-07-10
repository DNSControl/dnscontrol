## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `NETBIRD` along with a NetBird API token.

Example:

{% code title="creds.json" %}
```json
{
  "netbird": {
    "TYPE": "NETBIRD",
    "token": "your-netbird-api-token"
  }
}
```
{% endcode %}

## Metadata

This provider recognizes the following metadata fields:

| Key | Type | Value | Description |
|-------|------|---------|-------------|
| `netbird_enabled` | string | `"true"`/`"false"` |  Whether the zone is enabled. |
| `netbird_enable_search_domain` | string | `"true"`/`"false"` | Whether to enable this zone as a search domain. |

{% hint style="info" %}
**NOTE**: If metadata fields are not set, DNSControl will leave them unchanged in NetBird.
{% endhint %}

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NONE, DnsProvider(DSP_NETBIRD),
    { no_ns: "true" }, // NetBird does not expose nameservers
    A("test", "1.2.3.4"),
    AAAA("ipv6test", "2001:db8::1"),
    CNAME("www", "example.com"),
);
```
{% endcode %}

{% hint style="info" %}
**NOTE**: NetBird does not expose nameservers, so `{no_ns: "true"}` should be set on all domains to suppress the "Skipping registrar" warning.
{% endhint %}

To configure zone options, use metadata:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NONE,
    {
        no_ns: "true",
        netbird_enabled: "true",
        netbird_enable_search_domain: "true",
    },
    DnsProvider(DSP_NETBIRD),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation

NetBird depends on a NetBird API token. You can generate a personal access token in the NetBird dashboard.

## Caveats

NetBird API currently supports the following DNS record types:

- **A**
- **AAAA**
- **CNAME**

For more information, see the [NetBird API documentation](https://docs.netbird.io/api/resources/dns-zones).
