## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `GCORE`
along with a Gcore account API token.

Example:

{% code title="creds.json" %}
```json
{
  "gcore": {
    "TYPE": "GCORE",
    "api-key": "your-gcore-api-key"
  }
}
```
{% endcode %}

## Metadata
This provider supports the following metadata fields specific to Gcore, to support Gcore's GeoDNS and failover features.

All metadata values are a string, instead of a number/int or a boolean value. If you want to set a number or a boolean value, you must specify them in their string forms.

### Record level metadata
These metadata fields can be set on each individual record:

- `gcore_asn`: Comma separated string of ASNs the record should be served to.
- `gcore_continents`: Comma separated string of continents the record should be served to. Valid values are: `as,na,an,sa,oc,eu,af`.
- `gcore_countries`: Comma separated string of countries and regions the record should be served to. Countries and regions are represented by their two-letter code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
- `gcore_latitude` and `gcore_longitude`: Coordinate of the server's location the record is pointing to. Used for GeoDistance feature.
- `gcore_notes`: Arbitrary notes for the record.
- `gcore_weight`: Weight of the record, used in load balancing.
- `gcore_ip`: Comma separated string of IPs/CIDRs the record should be served to.

### Failover (Healthcheck) metadata
These metadata fields are shared within the same RRSet (record name and type combo). The failover metadata MUST be set on all the records within the RRSet, and MUST be exactly the same across all records.

- `gcore_failover_protocol`: Protocol to perform healthcheck on the server the record is pointing to. Valid values: `HTTP, TCP, UDP, ICMP`
- `gcore_failover_port`: The port (as a string) to connect to the server.
- `gcore_failover_frequency`: How often healthcheck should be performed, in seconds as a string. Valid values: 10-3600
- `gcore_failover_timeout`: How long healthcheck should wait for server's response, in seconds as a string. Valid values: 1-10
- `gcore_failover_method`: HTTP method used in the healthcheck. Only applies if `gcore_failover_protocol` is set to `HTTP`.
- `gcore_failover_command`: Bytes to be sent to the server in the healthcheck. Only applies if `gcore_failover_protocol` is set to `TCP` or `UDP`.
- `gcore_failover_url`: Relative URL to be requested, e.g. `/`. Only applies if `gcore_failover_protocol` is set to `HTTP`.
- `gcore_failover_tls`: If SSL/TLS should be used when connecting to origin server. Can be either `true` or `false` as a string.
- `gcore_failover_regexp`: Regular expression of expected contents in the response.
- `gcore_failover_http_status_code`: Expected HTTP status code as a string. Only applies if `gcore_failover_protocol` is set to `HTTP`.
- `gcore_failover_host`: Host field in the HTTP request header. Only applies if `gcore_failover_protocol` is set to `HTTP`.

### Filters

The `gcore_filters` metadata is a semicolon delimited string of several "filter" values.

Each filter is a comma separated string of 2 or 3 fields:

- 2 fields: `type,strict`.
- 3 fields: `type,strict,limit`.

As for the meaning of the fields:

- `type` is the type of the filter, e.g. `healthcheck`, `geodistance`.
- `strict` specifies what happens if the filter returns no records. If `strict` is `false`, then all records will be returned. If `strict` is `true`, then no records will be returned.
- `limit` specifies the maximum number of records to be returned. This is an optional field.

Let's take an example `gcore_filters` value: `healthcheck,false;geodistance,false;first_n,false,2`

This example filter specifies 3 filters: `healthcheck`, `geodistance`, `first_n`. For all three filters, the `strict` field is set to `false`. For `first_n` filter, the max number of records to return is 2.

### But this is too complicated for me! (Generating metadata with GCore itself)

GCore provider will also display the record metadata in the corrections list. If you're overwhelmed by the instructions above, you can do the following instead:

1. Add your records manually on Gcore's DNS control panel, and set all the metadata/filters you need.
2. Run `dnscontrol preview`. You will see DNSControl wants to delete your records:

   ```
   - DELETE test.example.com A 1.1.1.1 {"gcore_asn":"1234,2345","gcore_continents":"af,an,as,eu,na,oc,sa","gcore_countries":"cn,us","gcore_filters":"geodistance,false;first_n,false,2","gcore_ip":"1.2.3.4","gcore_latitude":"34.567","gcore_longitude":"12.89","gcore_notes":"test","gcore_weight":"12"} ttl=60
   ```

   Here DNSControl has generated all the metadata above based on your existing records.

3. Copy the metadata into your `dnsconfig.js` file.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_GCORE = NewDnsProvider("gcore");

D("example.com", REG_NONE, DnsProvider(DSP_GCORE),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

### Example with metadata

An example configuration with metadata set:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_GCORE = NewDnsProvider("gcore");

D("example.com", REG_NONE, DnsProvider(DSP_GCORE),
    A("@", "1.1.1.1", TTL("1m"), {
        gcore_filters: "geodistance,false;first_n,false,2",
        gcore_failover_protocol: "HTTP",
        gcore_failover_port: "443",
        gcore_failover_frequency: "30",
        gcore_failover_timeout: "10",
        gcore_failover_method: "GET",
        gcore_failover_url: "/",
        gcore_failover_tls: "true",
        gcore_failover_regexp: "",
        gcore_failover_host: "example.com",
        gcore_asn: "1234,2345",
        gcore_continents: "as,na,an,sa,oc,eu,af",
        gcore_countries: "cn,us",
        gcore_latitude: "12.345",
        gcore_longitude: "67.890",
        gcore_notes: "test",
        gcore_weight: "12",
        gcore_ip: "1.2.3.4",
    }),
    A("@", "1.1.1.2", TTL("1m"), {
        gcore_filters: "geodistance,false;first_n,false,2",
        gcore_failover_protocol: "HTTP",
        gcore_failover_port: "443",
        gcore_failover_frequency: "30",
        gcore_failover_timeout: "10",
        gcore_failover_method: "GET",
        gcore_failover_url: "/",
        gcore_failover_tls: "true",
        gcore_failover_regexp: "",
        gcore_failover_host: "example.com",
        gcore_asn: "1234,2345",
        gcore_continents: "as,na,an,sa,oc,eu,af",
        gcore_countries: "cn,us",
        gcore_latitude: "12.890",
        gcore_longitude: "34.567",
        gcore_notes: "test",
        gcore_weight: "34",
        gcore_ip: "1.2.3.5",
    }),
);
```
{% endcode %}

## Activation

DNSControl depends on a Gcore account API token.

You can obtain your API token on this page: <https://accounts.gcore.com/profile/api-tokens>

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
