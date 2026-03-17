{% hint style="info" %}
**NOTE**: This provider is currently has no maintainer. We are looking for
a volunteer. If this provider breaks it may be disabled or removed if
it can not be easily fixed.
{% endhint %}

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `SOFTLAYER`
along with authentication fields.
Authenticating with SoftLayer requires at least a `username` and `api_key` for authentication. It can also optionally take a `timeout` and `endpoint_url` parameter however these are optional and will use standard defaults if not provided.

Example:

{% code title="creds.json" %}
```json
{
  "softlayer": {
    "TYPE": "SOFTLAYER",
    "api_key": "mysecretapikey",
    "username": "myusername"
  }
}
```
{% endcode %}

To maintain compatibility with existing softlayer CLI services these can also be provided by the `SL_USERNAME` and `SL_API_KEY` environment variables or specified in the `~/.softlayer`, but this is discouraged. More information about these methods can be found at [the softlayer-go library documentation](https://github.com/softlayer/softlayer-go#sessions).

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_SOFTLAYER = NewDnsProvider("softlayer");

D("example.com", REG_NONE, DnsProvider(DSP_SOFTLAYER),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to SoftLayer dns.
For compatibility with the pre-generated NAMESERVER fields it's recommended to set the NS TTL to 86400 such as:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_SOFTLAYER = NewDnsProvider("softlayer");

D("example.com", REG_NONE, DnsProvider(SOFTLAYER),
    NAMESERVER_TTL(86400),

    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❌
  - get-zones: ❔
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
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
