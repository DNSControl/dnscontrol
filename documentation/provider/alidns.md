## Configuration

This provider is for [Alibaba Cloud DNS](https://www.alibabacloud.com/product/dns) (also known as ALIDNS). To use this provider, add an entry to `creds.json` with `TYPE` set to `ALIDNS` along with your API credentials.

Example:

{% code title="creds.json" %}
```json
{
  "alidns": {
    "TYPE": "ALIDNS",
    "access_key_id": "YOUR_ACCESS_KEY_ID",
    "access_key_secret": "YOUR_ACCESS_KEY_SECRET"
  }
}
```
{% endcode %}

Optionally, you can specify a `region_id`:

{% code title="creds.json" %}
```json
{
  "alidns": {
    "TYPE": "ALIDNS",
    "access_key_id": "YOUR_ACCESS_KEY_ID",
    "access_key_secret": "YOUR_ACCESS_KEY_SECRET",
    "region_id": "cn-hangzhou"
  }
}
```
{% endcode %}

Note: The `region_id` defaults to `"cn-hangzhou"`. The region value does not affect DNS management (DNS is global), but Alibaba's SDK requires a region to be provided.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_ALIDNS = NewDnsProvider("alidns");

D("example.com", REG_NONE, DnsProvider(DSP_ALIDNS),
    A("test", "1.2.3.4"),
    CNAME("www", "example.com."),
    MX("@", 10, "mail.example.com."),
);
```
{% endcode %}

## Activation

DNSControl depends on an Alibaba Cloud [RAM user](https://www.alibabacloud.com/help/en/ram/user-guide/overview-of-ram-users) with permissions to manage DNS records.

### Creating RAM User and Access Keys

1. Log in to the [RAM console](https://ram.console.aliyun.com/)
2. Create a new RAM user or use an existing one
3. Generate an AccessKey ID and AccessKey Secret for the user
4. Attach the `AliyunDNSFullAccess` policy to the user

The minimum required permissions are:

```json
{
  "Version": "1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "alidns:DescribeDomains",
        "alidns:DescribeDomainRecords",
        "alidns:DescribeDomainInfo",
        "alidns:AddDomainRecord",
        "alidns:UpdateDomainRecord",
        "alidns:DeleteDomainRecord"
      ],
      "Resource": "*"
    }
  ]
}
```

## Important Notes

### TTL Constraints

Alibaba Cloud DNS has different TTL constraints depending on your DNS edition:

- **Enterprise Ultimate Edition**: TTL can be as low as 1 second (1-86400)
- **Personal Edition / Free Edition**: Minimum TTL is 600 seconds (600-86400)

DNSControl will automatically validate TTL values based on your domain's edition. If you attempt to use a TTL below the minimum for your edition, you will receive an error.

### Chinese Domain Name Support

ALIDNS supports Chinese domain names (IDN with Chinese characters). However:

- **Supported**: ASCII characters and Chinese characters (CJK Unified Ideographs)
- **Not supported**: Other Unicode characters (e.g., German umlauts, Arabic script)

DNSControl will automatically convert between punycode and unicode as needed.

### Record Type Support

The following record types are supported:
- A, AAAA, CNAME, MX, TXT, NS
- CAA (requires quoted values: `0 issue "letsencrypt.org"`)
- SRV

### TXT Record Constraints

Alibaba Cloud DNS has specific constraints for TXT records:
- Cannot be empty
- Maximum length: 512 bytes
- Cannot contain unescaped double quotes
- Cannot have trailing spaces
- Cannot have unpaired backslashes (odd number of consecutive backslashes)

DNSControl will audit and reject records that violate these constraints.

## New Domains

If a domain does not exist in your Alibaba Cloud account, you must create it manually through the Alibaba Cloud console. DNSControl does not automatically create new domains for this provider.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ❌
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
