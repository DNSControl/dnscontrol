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

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❌
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❌
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
