## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `NETNOD` along with your API URL and API Key. The API URL can be omitted to use the default value `https://primarydnsapi.netnod.se`.

Example:

{% code title="creds.json" %}

```json
{
    "netnod": {
        "TYPE": "NETNOD",
        "apiKey": "your-key",
        "apiUrl": "https://primarydnsapi.netnod.se"
    }
}
```

{% endcode %}

## Metadata

The following provider metadata is available:

{% code title="dnsconfig.js" %}

```javascript
var DSP_NETNOD = NewDnsProvider('netnod', {
    default_ns: ['a.example.com.', 'b.example.com.'],
    also_notify: ['192.36.148.17', '2001:7fe::53'],
    allow_transfer_keys: ['netnod-key1.'],
});
```

{% endcode %}

- `default_ns` sets the nameservers used when creating zones.
- `also_notify` sets a list of IP addresses that will receive DNS NOTIFY messages when a zone is created. This is the provider-level default and applies to all zones unless overridden per zone.
- `allow_transfer_keys` sets the TSIG key IDs permitted to perform zone transfers from the distribution servers when a zone is created.
  This should include all keys used for DNS secondary replication, including those used by the Netnod secondary DNS service. This is the provider-level default and applies to all zones unless overridden per zone.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}

```javascript
var REG_NONE = NewRegistrar('none');
var DSP_NETNOD = NewDnsProvider('netnod');

D('example.com', REG_NONE, DnsProvider(DSP_NETNOD), A('test', '1.2.3.4'));
```

{% endcode %}

## Activation

See the [Netnod DNS](https://www.netnod.se/dns/dns-enterprise-services).
