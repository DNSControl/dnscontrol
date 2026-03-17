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
var REG_CSCGLOBAL = NewRegistrar("cscglobal");
var DSP_BIND = NewDnsProvider("bind");

D("example.com", REG_CSCGLOBAL, DnsProvider(DSP_BIND),
  A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
To get access to the [CSC Global API](https://www.cscglobal.com/cscglobal/docs/dbs/domainmanager/api-v2/) contact your account manager.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ✅
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❌
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
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
