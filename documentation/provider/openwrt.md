This is the provider for [OpenWrt](https://openwrt.org/).

To use this provider you need to install the `luci-mod-rpc`package on the OpenWrt instance.

## Important notes

This provider only supports the following record types.

* [A](../language-reference/domain-modifiers/A.md)
* [AAAA](../language-reference/domain-modifiers/AAAA.md)
* [CNAME](../language-reference/domain-modifiers/CNAME.md)
* [MX](../language-reference/domain-modifiers/MX.md)
* [SRV](../language-reference/domain-modifiers/SRV.md)

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `OPENWRT`.

Required fields include:

* `username` and `password`: Authentication information
* `host`: The hostname/address of OpenWrt instance

Example:

{% code title="creds.json" %}
```json
{
  "openwrt": {
    "TYPE": "OPENWRT",
    "username": "root",
    "password": "your-password",
    "host": "http://192.168.1.1"
  }
}
```
{% endcode %}

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_OPENWRT = NewDnsProvider("openwrt");

D("example.com", REG_NONE, DnsProvider(DSP_OPENWRT),
    A("foo", "1.2.3.4"),
    AAAA("another", "2003::1"),
    CNAME("myalias", "www.example.com."),
    MX("@", 5, "mail"),
    SRV("_sip._tcp", 10, 60, 5060, "pbx.example.com."),
);
```
{% endcode %}

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❔
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❔
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ❔
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❔
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❔
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
