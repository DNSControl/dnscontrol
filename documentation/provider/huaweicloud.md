## Configuration


This provider is for the [Huawei Cloud DNS](https://www.huaweicloud.com/intl/en-us/product/dns.html)(Public DNS).  To use this provider, add an entry to `creds.json` with `TYPE` set to `HUAWEICLOUD`.
along with the API credentials.

Example:

{% code title="creds.json" %}
```json
{
  "huaweicloud": {
    "TYPE": "HUAWEICLOUD",
    "KeyId": "YOUR_ACCESS_KEY_ID",
    "SecretKey": "YOUR_SECRET_ACCESS_KEY",
    "Region": "YOUR_SERVICE_REGION"
  }
}
```
{% endcode %}

## Metadata
There are some record level metadata available for this provider:
   * `hw_line` (Line ID, default "default_view") Refer to the [Intelligent Resolution](https://support.huaweicloud.com/intl/en-us/usermanual-dns/dns_usermanual_0041.html) for more information.
     * Available Line ID refer to [Resolution Lines](https://support.huaweicloud.com/intl/en-us/api-dns/en-us_topic_0085546214.html). Custom Line ID can also be used.
   * `hw_weight` (0-1000, default "1") Refer to the [Configuring Weighted Routing](https://support.huaweicloud.com/intl/en-us/usermanual-dns/dns_usermanual_0705.html) for more information.
   * `hw_rrset_key` (default "") User defined key for RRset load balance. This value would be stored in the description field of the RRset.

The following example shows how to use the metadata:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_HWCLOUD = NewDnsProvider("huaweicloud");

D("example.com", REG_NONE, DnsProvider(DSP_HWCLOUD),
    // this example will create 4 rrsets with the same name "test"
    A("test", "8.8.8.8"),
    A("test", "8.8.4.4"),
    A("test", "9.9.9.9", {hw_weight: "10"}),         // Weighted Routing
    A("test", "149.112.112.112", {hw_weight: "10"}), // Weighted Routing
    A("test", "223.5.5.5", {hw_line: "CN"}), // GEODNS
    A("test", "223.6.6.6", {hw_line: "CN", hw_weight: "10"}), // GEODNS with weight

    // this example will create 3 rrsets with the same name "rr-lb"
    A("rr-lb", "10.0.0.1", {hw_weight: "10", hw_rrset_key: "lb-zone-a"}),
    A("rr-lb", "10.0.0.2", {hw_weight: "10", hw_rrset_key: "lb-zone-a"}),
    A("rr-lb", "10.0.1.1", {hw_weight: "10", hw_rrset_key: "lb-zone-b"}),
    A("rr-lb", "10.0.1.2", {hw_weight: "10", hw_rrset_key: "lb-zone-b"}),
    A("rr-lb", "10.0.2.2", {hw_weight: "0",  hw_rrset_key: "lb-zone-c"}),
);
```
{% endcode %}

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_HWCLOUD = NewDnsProvider("huaweicloud");

D("example.com", REG_NONE, DnsProvider(DSP_HWCLOUD),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
DNSControl depends on a standard [IAM User](https://support.huaweicloud.com/intl/en-us/usermanual-iam/iam_02_0003.html) with permission to list, create and update hosted zones.

The `DNS FullAccess` policy will also work, but that provides access to many other areas and violates the "principle of least privilege".

The minimum permissions required are as follows:

```json
{
    "Version": "1.1",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "dns:recordset:delete",
                "dns:recordset:create",
                "dns:zone:create",
                "dns:recordset:get",
                "dns:nameserver:getZoneNameServer",
                "dns:zone:list",
                "dns:recordset:update",
                "dns:recordset:list",
                "dns:zone:get",
                "dns:zone:getDnssecConfig",
                "dns:zone:enableDnssecConfig",
                "dns:zone:disableDnssecConfig"
            ]
        }
    ]
}
```

To determine the `Region` parameter, refer to the [endpoint page of huaweicloud](https://console-intl.huaweicloud.com/apiexplorer/#/endpoint/DNS). For example, on the international site, the `Region` name `ap-southeast-1` is known to work.

If that doesn't work, log into Huaweicloud's website and open the [API Explorer](https://console-intl.huaweicloud.com/apiexplorer/#/openapi/DNS/debug?api=ListPublicZones), find the `ListPublicZones` API, select a different Region and click Debug to try and find your Region.

## New domains
If a domain does not exist in your Huawei Cloud account, DNSControl will automatically add it with the `push` command.

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
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❌
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❌
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
