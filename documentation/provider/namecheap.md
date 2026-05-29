This is the provider for [Namecheap](https://www.namecheap.com/).

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `NAMECHEAP` along with your Namecheap API username and key:

Example:

{% code title="creds.json" %}
```json
{
  "namecheap": {
    "TYPE": "NAMECHEAP",
    "apikey": "yourApiKeyFromNameCheap",
    "apiuser": "yourUsername"
  }
}
```
{% endcode %}

You can optionally specify BaseURL to use a different endpoint - typically the sandbox:

{% code title="creds.json" %}
```json
{
  "namecheapSandbox": {
    "TYPE": "NAMECHEAP",
    "apikey": "yourApiKeyFromNameCheap",
    "apiuser": "yourUsername",
    "BaseURL": "https://api.sandbox.namecheap.com/xml.response"
  }
}
```
{% endcode %}

if BaseURL is omitted, the production namecheap URL is assumed.

## Metadata
This provider does not recognize any special metadata fields unique to Namecheap.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NAMECHEAP = NewRegistrar("namecheap");
var DSP_BIND = NewDnsProvider("bind");

D("example.com", REG_NAMECHEAP, DnsProvider(DSP_BIND),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

Namecheap provides custom redirect records URL, URL301, and FRAME.  These records can be used like any other record:

{% code title="dnsconfig.js" %}
```javascript
var REG_NAMECHEAP = NewRegistrar("namecheap");
var DSP_NAMECHEAP = NewDnsProvider("namecheap");

D("example.com", REG_NAMECHEAP, DnsProvider(DSP_NAMECHEAP),
  URL("@", "http://example.com/"),
  URL("www", "http://example.com/"),
  URL301("backup", "http://backup.example.com/"),
);
```
{% endcode %}

## Activation
In order to activate API functionality on your Namecheap account, you must enable it for your account and wait for their review process. More information on enabling API access is [located here](https://www.namecheap.com/support/api/intro.aspx).

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❔
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ❌
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❔
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
