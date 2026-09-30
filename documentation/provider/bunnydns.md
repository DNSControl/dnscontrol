# Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `BUNNY_DNS` along with
your [Bunny API Key](https://dash.bunny.net/account/settings).

Example:

{% code title="creds.json" %}
```json
{
  "bunny_dns": {
    "TYPE": "BUNNY_DNS",
    "api_key": "your-bunny-api-key"
  }
}
```
{% endcode %}

You can also use environment variables:

```shell
export BUNNY_DNS_API_KEY=XXXXXXXXX
```

{% code title="creds.json" %}
```json
{
  "bunny_dns": {
    "TYPE": "BUNNY_DNS",
    "api_key": "$BUNNY_DNS_API_KEY"
  }
}
```
{% endcode %}

## Metadata

This provider supports the following metadata fields, used to configure Bunny DNS smart routing (geographic and latency-based routing) for `A` and `AAAA` records.

All metadata values are strings. Numeric values must be specified in their string forms.

### Smart routing metadata

These metadata fields can be set on individual `A` or `AAAA` records:

- `bunny_smart_routing_type`: The smart routing type. Valid values are:
  - `geographic` (or `geo`): Route queries based on the end user's geographical location. The record closest to the user is returned.
  - `latency`: Route queries based on estimated latency to the Bunny.net datacenter region closest to your server.
  - `none`: Disable smart routing (default; does not need to be specified).
- `bunny_geolocation_latitude`: The latitude coordinate of the server's location, as a string. Only used when `bunny_smart_routing_type` is `geographic`.
- `bunny_geolocation_longitude`: The longitude coordinate of the server's location, as a string. Only used when `bunny_smart_routing_type` is `geographic`.
- `bunny_latency_zone`: The Bunny.net datacenter region code closest to your server, e.g. `NY`. Only used when `bunny_smart_routing_type` is `latency`.

  The valid region codes can be retrieved from Bunny's public region list API, which does not require an API key:

  ```shell
  curl https://api.bunny.net/region
  ```

  This returns a JSON array of region objects. The `RegionCode` field is the value to use for `bunny_latency_zone`. Only regions where `AllowLatencyRouting` is `true` are valid for latency routing. For example:

  ```json
  {
    "Id": 2,
    "Name": "NA: New York City, NY",
    "RegionCode": "NY",
    "ContinentCode": "NA",
    "CountryCode": "US",
    "AllowLatencyRouting": true
  }
  ```

  See the [Region list API documentation](https://docs.bunny.net/api-reference/core/region/region-list) for full details.

For more information, see the [Bunny DNS Smart Records documentation](https://docs.bunny.net/dns/records#smart-records).

### Health monitoring metadata

Bunny DNS provides built-in health monitoring for `A`, `AAAA`, and `CNAME` records. When monitoring is enabled, each IP is tested every 30 seconds from 3 regions around the world. In a record set, offline records are automatically removed from responses.

- `bunny_monitor_type`: The monitoring type. Valid values are:
  - `ping`: Sends 4 ICMP ping packets to the monitored IP. At least one packet must return successfully within 2500ms.
  - `http`: Sends HTTP requests to the configured IP on port 80. The request must complete within 10 seconds and return a non-5xx status code.
  - `none`: Disable monitoring (default; does not need to be specified).

For more information, see the [Bunny DNS Monitoring documentation](https://docs.bunny.net/dns/records#health-monitoring).

### Example with geographic routing

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NONE, DnsProvider(DSP_BUNNY_DNS),
    A("www", "1.2.3.4", {
        bunny_smart_routing_type: "geographic",
        bunny_geolocation_latitude: "40.7128",
        bunny_geolocation_longitude: "-74.0060",
    }),
    A("www", "5.6.7.8", {
        bunny_smart_routing_type: "geographic",
        bunny_geolocation_latitude: "48.8566",
        bunny_geolocation_longitude: "2.3522",
    }),
);
```
{% endcode %}

### Example with latency routing

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NONE, DnsProvider(DSP_BUNNY_DNS),
    A("www", "1.2.3.4", {
        bunny_smart_routing_type: "latency",
        bunny_latency_zone: "NY",
    }),
    A("www", "5.6.7.8", {
        bunny_smart_routing_type: "latency",
        bunny_latency_zone: "FRA",
    }),
);
```
{% endcode %}

### Example with health monitoring

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NONE, DnsProvider(DSP_BUNNY_DNS),
    A("www", "1.2.3.4", {
        bunny_monitor_type: "ping",
    }),
    A("www", "5.6.7.8", {
        bunny_monitor_type: "http",
    }),
);
```
{% endcode %}

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_BUNNY_DNS = NewDnsProvider("bunny_dns");

D("example.com", REG_NONE, DnsProvider(DSP_BUNNY_DNS),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

# Activation

DNSControl depends on the [Bunny API](https://docs.bunny.net/reference/bunnynet-api-overview) to manage your DNS
records. You will need to generate an [API key](https://dash.bunny.net/account/settings) to use this provider.

## New domains

If a domain does not exist in your Bunny account, DNSControl will automatically add it with the `push` command.

## Custom record types

DNSControl supports only the custom record types listed below for Bunny DNS. Other Bunny-specific types
(such as Script or Flatten) are not supported and will be ignored by DNSControl and left as-is.

### Redirect

You can configure Bunny's Redirect type with `BUNNY_DNS_RDR`:

{% code title="dnsconfig.js" %}
```javascript
    BUNNY_DNS_RDR("@", "https://foo.bar"),
```
{% endcode %}

### Pull Zone (PZ)

You can configure Bunny's Pull Zone type with `BUNNY_DNS_PZ`. The target is the Pull Zone ID:

{% code title="dnsconfig.js" %}
```javascript
    BUNNY_DNS_PZ("@", 12345),
```
{% endcode %}

## Caveats

- Bunny DNS does not support dual-hosting or configuring custom TTLs for NS records on the zone apex.
- While custom nameservers are properly recognized by this provider, it is currently not possible to configure them.

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
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❌
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
