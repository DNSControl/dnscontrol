This is the provider for [Sakura Cloud](https://cloud.sakura.ad.jp/).

## Configuration
To use this provider, add an entry to `creds.json` with `TYPE` set to `SAKURACLOUD` along with API credentials.

Example:

{% code title="creds.json" %}
```json
{
  "sakuracloud": {
    "TYPE": "SAKURACLOUD",
    "access_token": "your-access-token",
    "access_token_secret": "your-access-token-secret"
  }
}
```
{% endcode %}

The `endpoint` is optional. If omitted, the default endpoint is assumed.

Endpoints are as follows:

* `https://secure.sakura.ad.jp/cloud/zone/is1a/api/cloud/1.1` (Ishikari first Zone)
* `https://secure.sakura.ad.jp/cloud/zone/is1b/api/cloud/1.1` (Ishikari second Zone)
* `https://secure.sakura.ad.jp/cloud/zone/tk1a/api/cloud/1.1` (Tokyo first Zone)
* `https://secure.sakura.ad.jp/cloud/zone/tk1b/api/cloud/1.1` (Tokyo second Zone)

DNS service is independent of zones, so you can use any of these endpoints. The default is the Ishikari first Zone.

Alternatively you can also use environment variables.

```shell
export SAKURACLOUD_ACCESS_TOKEN="your-access-token"
export SAKURACLOUD_ACCESS_TOKEN_SECRET="your-access-token-secret"
```

{% code title="creds.json" %}
```json
{
  "sakuracloud": {
    "TYPE": "SAKURACLOUD",
    "access_token": "$SAKURACLOUD_ACCESS_TOKEN",
    "access_token_secret": "$SAKURACLOUD_ACCESS_TOKEN_SECRET"
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to Sakura Cloud.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_SAKURACLOUD = NewDnsProvider("sakuracloud");

D("example.com", REG_NONE, DnsProvider(DSP_SAKURACLOUD),
  A("test", "192.0.2.1"),
);
```
{% endcode %}

`NAMESERVER` does not need to be set as the name servers for the Sakura Cloud provider cannot be changed.

`SOA` cannot be set as SOA record of Sakura Cloud provider cannot be changed.

## Activation
Sakura Cloud depends on an [API Key](https://manual.sakura.ad.jp/cloud/api/apikey.html).

When creating an API key, select "can modify settings" as "Access level". if you plan to create zones, select "can create and delete resources" as "Access level". None of the options in the "Allow access to other services" field need to be checked.

## Caveats
The limitations of the Sakura Cloud DNS service are described in [the DNS manual](https://manual.sakura.ad.jp/cloud/appliance/dns/index.html), which is written in Japanese.

The limitations not described in that manual are:

* "Null MX", RFC 7505, is not supported.
* SRV records with a Target of "." are not supported.
* SRV records with Port "0" are not supported.
* CAA records with a property value longer than 64 bytes are not allowed.
* Owner names and RDATA targets containing the following labels are not allowed:
    * example
    * exampleN, where N is a numerical character

## Feature Flags

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
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❌
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
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
