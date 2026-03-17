## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `PORKBUN`
along with your `api_key` and `secret_key`. More info about authentication can be found in [Getting started with the Porkbun API](https://kb.porkbun.com/article/190-getting-started-with-the-porkbun-api).

Example:

{% code title="creds.json" %}
```json
{
  "porkbun": {
    "TYPE": "PORKBUN",
    "api_key": "your-porkbun-api-key",
    "secret_key": "your-porkbun-secret-key"
  }
}
```
{% endcode %}

Porkbun has quite strict API limits. If you experience errors with this provider (common when you have many domains), you can set one or both of `max_attempts` and `max_duration` in the credentials configuration.

Example:

{% code title="creds.json" %}
```json
{
  "porkbun": {
    "TYPE": "PORKBUN",
    "api_key": "your-porkbun-api-key",
    "secret_key": "your-porkbun-secret-key",
    "max_attempts": "10",
    "max_duration": "5m"
  }
}
```
{% endcode %}

The default for `max_attempts` is 5. There is no maximum duration by default, instead the provider will perform exponential backoff between 1 and 10 seconds, until `max_attempts` is reached. To retry indefinitely until `max_duration` is reached, set `max_attempts` to any value below 1.

## Metadata

This provider does not recognize any special metadata fields unique to Porkbun.

## Usage

An example configuration: (DNS hosted with Porkbun):

{% code title="dnsconfig.js" %}
```javascript
var REG_PORKBUN = NewRegistrar("porkbun");
var DSP_PORKBUN = NewDnsProvider("porkbun");

D("example.com", REG_PORKBUN, DnsProvider(DSP_PORKBUN),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

An example configuration: (Registrar only. DNS hosted elsewhere)

{% code title="dnsconfig.js" %}
```javascript
var REG_PORKBUN = NewRegistrar("porkbun");
var DSP_R53 = NewDnsProvider("r53");

D("example.com", REG_PORKBUN, DnsProvider(DSP_R53),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## URL Forwarding

Porkbun supports URL forwarding (redirects) using the `URL` and `URL301` record types:

{% code title="dnsconfig.js" %}
```javascript
var REG_PORKBUN = NewRegistrar("porkbun");
var DSP_PORKBUN = NewDnsProvider("porkbun");

D("example.com", REG_PORKBUN, DnsProvider(DSP_PORKBUN),
    // Temporary redirect (HTTP 302)
    URL("redirect", "https://example.org"),
    // Permanent redirect (HTTP 301)
    URL301("www", "https://example.com"),
);
```
{% endcode %}

By default, URL forwarding includes wildcard subdomains but does not include the URI path in redirection. You can customize this behavior using metadata:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_PORKBUN, DnsProvider(DSP_PORKBUN),
    // Include path and disable wildcard
    URL("redirect", "https://example.org", {includePath: "yes", wildcard: "no"}),
);
```
{% endcode %}

{% hint style="info" %}
**NOTE**: The legacy `PORKBUN_URLFWD` record type is deprecated. Please use `URL` or `URL301` instead.
{% endhint %}

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
