DNSControl's CSC Global provider supports being a Registrar. Support for being a DNS Provider is not included, although CSC Global's API does provide for this so it could be implemented in the future.

{% hint style="info" %}
**NOTE**: Experimental support for being a DNS Provider is available.
However it is not recommended as updates take 5-7 minutes, and the
next update is not permitted until the previous update is complete.
Use it at your own risk.  Consider it experimental and undocumented.
{% endhint %}

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `CSCGLOBAL`.

In your `creds.json` file, you must provide your API key and user/client token. You can optionally provide an comma separated list of email addresses to have CSC Global send updates to.

Example:

{% code title="creds.json" %}
```json
{
  "cscglobal": {
    "TYPE": "CSCGLOBAL",
    "api-key": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "user-token": "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy",
    "notification_emails": "test@example.com,hostmaster@example.com"
  }
}
```
{% endcode %}

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("cscglobal"), SERVICE("bind"),
  A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
To get access to the [CSC Global API](https://www.cscglobal.com/cscglobal/docs/dbs/domainmanager/api-v2/) contact your account manager.

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ✅
  - DNS Provider: ✅
  - Registrar: ✅
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
