## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `POWERDNS`
along with your [API URL, API Key and Server ID](https://doc.powerdns.com/authoritative/http-api/index.html).
In most cases the Server id (`serverName`) is `localhost`.

Example:

{% code title="creds.json" %}
```json
{
  "powerdns": {
    "TYPE": "POWERDNS",
    "apiKey": "your-key",
    "apiUrl": "http://localhost",
    "serverName": "localhost"
  }
}
```
{% endcode %}

## Metadata
Following provider metadata are available:

{% code title="dnsconfig.js" %}
```javascript
var DSP_POWERDNS = NewDnsProvider("pdns", {
    "default_ns": [
        "a.example.com.",
        "b.example.com."
    ],
    "dnssec_on_create": false,
    "zone_kind": "Native",
    "use_views": true
});
```
{% endcode %}

- `default_ns` sets the nameservers which are used.
- `dnssec_on_create` specifies if DNSSEC should be enabled when creating zones.
- `zone_kind` is the type that will be used when creating the zone.
  <br>Can be one of `Native`, `Master` or `Slave`, when not specified it defaults to `Native`.
  <br>Please see [PowerDNS documentation](https://doc.powerdns.com/authoritative/modes-of-operation.html) for explanation of the kinds.
  <br>**Note that these tokens are case-sensitive!**
- `soa_edit_api` is the default SOA serial method that is used for zone created with the API
  <br> Can be one of `DEFAULT`, `INCREASE`, `EPOCH`, `SOA-EDIT` or `SOA-EDIT-INCREASE`, default format is YYYYMMDD01.
  <br>Please see [PowerDNS SOA-EDIT-DNSUPDATE documentation](https://doc.powerdns.com/authoritative/dnsupdate.html#soa-edit-dnsupdate-settings) for explanation of the kinds.
  <br>**Note that these tokens are case-sensitive!**
- `use_views` enables mapping dnscontrol tags to PowerDNS views.
  <br>Set to `true` to enable, defaults to `false`.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_POWERDNS = NewDnsProvider("powerdns");

D("example.com", REG_NONE, DnsProvider(DSP_POWERDNS),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
See the [PowerDNS documentation](https://doc.powerdns.com/authoritative/http-api/index.html) how the API can be enabled.

## HTTPS/SVCB Automatic Hints

PowerDNS supports automatic address hints for `HTTPS` and `SVCB` records. DNSControl preserves the PowerDNS-specific `ipv4hint=auto` and `ipv6hint=auto` parameters when using this provider:

{% code title="dnsconfig.js" %}
```javascript
HTTPS("foo", 1, ".", "alpn=h3,h2 ipv4hint=auto ipv6hint=auto");
```
{% endcode %}

## Tags and Variants
If you use a dnscontrol *tag* (like `example.com!internal`) it will be mapped to a powerdns *variant* (like `example.com..internal`) when `use_views` is enabled in the provider metadata.

See [PowerDNS documentation on Views](https://doc.powerdns.com/authoritative/views.html) for details on how to setup networks and views for these variants.

## Caveats

### SOA Records
SOA-support was implemented in version 4.16. Versions earlier than that (e.g., 4.15 and earlier) do not support SOA records and will raise an error if they are present in DNSControl. However, version 4.16 _requires_ SOA-records to be present in DNSControl, since DNSControl will synchronize SOA records as any normal records to PowerDNS (as PowerDNS handles SOA as any other record), and thus remove them if not present in DNSControl. See below for tips how to handle this.

In version later than 4.16+ the SOA record is supported for use, but behavior is slightly different than expected.
If the SOA record is used, [PowerDNS will not increase the serial](https://doc.powerdns.com/authoritative/dnsupdate.html#soa-serial-updates) if the SOA record content changes.
This itself comes with exceptions as well, if the `SOA-EDIT-API` is changed to a different value the logic will update the serial to a new value.
See [this issue for detailed testing](https://github.com/DNSControl/dnscontrol/pull/3404#issuecomment-2628989200) of behavior.

The recommended procedure when changing the SOA record contents is to update the SOA record alone.
Updates to other records will be done if changes are present, but the serial **will not change**. The serial will update once a new push is done that does not include an SOA record change.

### Tips for upgrading past version 4.16
Since dnscontrol v4.16 SOA-records have to be present in dnscontrol for PowerDNS. This is a breaking change from version 4.15 and requires changes on the user side. 

If you have a large number of zones it might be useful to handle this via built-in functions of dnscontrol. 

{% code title="dnsconfig.js" %}
```javascript

// Default SOA
var SOA_DEFAULT = [
  SOA(
  "@",
  "ns.example.org.", // <-- Change to your nameserver
  "noc.example.org.", // <-- Change to your contact address / administrators address
  7200, // <--refresh
  900, // <--retry
  604800, // <--expire
  1800, // <-- ttl for _ZONE_
  TTL("1h")) // <-- ttl for _RECORD_
];

// Add default SOA to all configured domains
var domains = getConfiguredDomains();
for (i = 0; i < domains.length; i++) {
  // Possibly introduce an if-statment to set different SOAs to reverses, k8s-zones, etc
  D_EXTEND(domains[i], SOA_DEFAULT);
}
```
{% endcode %}

This will set a default SOA for all zones managed by this dnscontrol
instance. Note that you might want to have different SOAs for different
zones, for example a very low `ttl` for Kubernetes (K8s) managed zones, this can
be handled with an `if`-statement in the for loop. 

## Feature Flags

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
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ✅
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ✅
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ✅
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
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ✅
  - [`DS`](../language-reference/domain-modifiers/DS.md): ✅
<!-- provider-features-end -->
