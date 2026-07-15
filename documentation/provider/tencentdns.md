## Configuration

{% hint style="info" %}
This provider is developed for the **Tencent Cloud API 3.0** platform.
{% endhint %}

This provider is for [Tencent Cloud DNS](https://cloud.tencent.com/product/dns) (DNSPod). To use this provider, add an entry to `creds.json` with `TYPE` set to `TENCENTDNS` along with your [API secrets](https://console.intl.cloud.tencent.com/cam/capi).

Example:

{% code title="creds.json" %}
```json
{
  "tencentdns": {
    "TYPE": "TENCENTDNS",
    "secret_id": "YOUR_SECRET_ID",
    "secret_key": "YOUR_SECRET_KEY",
    "site": "cn | intl"
  }
}
```
{% endcode %}

Optionally, you can specify a `region` (defaults to `"ap-guangzhou"`):

{% code title="creds.json" %}
```json
{
  "tencentdns": {
    "TYPE": "TENCENTDNS",
    "secret_id": "YOUR_SECRET_ID",
    "secret_key": "YOUR_SECRET_KEY",
    "region": "ap-guangzhou",
    "site": "intl"
  }
}
```
{% endcode %}

Optionally, you can specify a `site` (defaults to `"cn"`). Use `"intl"` for Tencent Cloud International accounts:

{% code title="creds.json" %}
```json
{
  "tencentdns": {
    "TYPE": "TENCENTDNS",
    "secret_id": "YOUR_SECRET_ID",
    "secret_key": "YOUR_SECRET_KEY",
    "site": "intl"
  }
}
```
{% endcode %}

Valid `site` values are:

- `cn`: Tencent Cloud mainland China APIs.
- `intl`: Tencent Cloud International APIs.

The `site` setting affects both DNSPod DNS management and registrar nameserver updates.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_TENCENT = NewRegistrar("tencentdns", "TENCENTDNS");
var DSP_TENCENT = NewDnsProvider("tencentdns", "TENCENTDNS");

D("example.com", REG_TENCENT, DnsProvider(DSP_TENCENT),
    A("@", "1.2.3.4"),
    CNAME("www", "example.com."),
    MX("@", 10, "mail.example.com."),
    TXT("test", "hello world")
);
```
{% endcode %}

### Why use `ALIAS` for DNSPod

DNSPod does not natively support the `ALIAS` record type.

In this provider, `ALIAS("@")` is used only as a DNSControl-side representation of CNAME flattening at the zone apex (`@`). It does not mean DNSPod has a real ALIAS record type.

We use `ALIAS("@")` because DNSControl treats `CNAME("@")` as invalid. In standard DNS, a CNAME record cannot be placed at the zone apex, because the apex already contains required records such as `SOA` and `NS`.

For DNSPod, the provider maps `ALIAS("@")` to a CNAME record on `@` under the hood. The actual CNAME flattening behavior must still be configured manually in the DNSPod dashboard.

#### Example:

**Recommended**

Use `ALIAS("@")` for apex CNAME flattening:

```js
D("example.com", REG_NONE, DnsProvider(DNSPOD),
  ALIAS("@", "target.example.net.")
);
```
**Not recommended**

Avoid writing CNAME("@") directly:

```js
D("example.com", REG_NONE, DnsProvider(DNSPOD),
  CNAME("@", "target.example.net.")
);
```

For compatibility, the DNSPod provider automatically converts apex CNAME("@") to ALIAS("@") internally. This allows DNSControl to treat it as an apex-flattening record instead of a standard apex CNAME.

### Note

DNSPod does not natively support the ALIAS record type. In this provider, ALIAS("@") is only a DNSControl-side representation of apex CNAME flattening.

When pushed to DNSPod, it is stored as a CNAME record on @.

Reference: https://docs.dnspod.com/dns/faq-dns-resolution/?lang=en


## Important Notes

### Features

- **MX Records**: Priority and target are handled automatically.
- **Registrar Support**: Supports updating authoritative nameservers for domains registered with Tencent Cloud.
- **Tencent Cloud Site**: Use `site: "intl"` for Tencent Cloud International site, use `site: "cn"` for Tencent Cloud China site.
- **Line Management**: All records are created on the "默认" (Default) line.
- **New Domains**: DNSControl will automatically create non-existent domains in your account.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❔
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❔
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❔
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
