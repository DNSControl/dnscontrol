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
D("example.com", REGISTRAR("namecheap"), SERVICE("bind"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

Namecheap provides custom redirect records URL, URL301, and FRAME.  These records can be used like any other record:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("namecheap"), SERVICE("namecheap"),
  URL("@", "http://example.com/"),
  URL("www", "http://example.com/"),
  URL301("backup", "http://backup.example.com/"),
);
```
{% endcode %}

## Activation
In order to activate API functionality on your Namecheap account, you must enable it for your account and wait for their review process. More information on enabling API access is [located here](https://www.namecheap.com/support/api/intro.aspx).

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
