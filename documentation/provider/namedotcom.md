{% hint style="info" %}
**NOTE**: This provider is currently has no maintainer. We are looking for
a volunteer. If this provider breaks it may be disabled or removed if
it can not be easily fixed.
{% endhint %}

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `NAMEDOTCOM` along with your name.com API username and access token:

Example:

{% code title="creds.json" %}
```json
{
  "name.com": {
    "TYPE": "NAMEDOTCOM",
    "apikey": "yourApiKeyFromName.com",
    "apiuser": "yourUsername"
  }
}
```
{% endcode %}

There is another key name `apiurl` but it is optional and defaults to the correct value. If you want to use the test environment ("OT&E"), then add this:

    "apiurl": "https://api.dev.name.com",

For compatibility with existing configurations, `apiurl` also accepts a hostname without `https://`.

## Metadata
This provider does not recognize any special metadata fields unique to name.com.

## Usage

An example `dnsconfig.js` configuration with NAMEDOTCOM as the registrar and DNS service provider:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("name.com"), SERVICE("name.com"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

An example `dnsconfig.js` configuration with NAMEDOTCOM as the registrar and DNS only, DNS hosted elsewhere:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("name.com"), SERVICE("r53"),
    A("test","1.2.3.4"),
);
```
{% endcode %}

{% hint style="info" %}
**NOTE**: name.com does not allow control over the NS records of your zones via the api. It is not recommended to use name.com's dns provider unless it is your only dns host.
{% endhint %}

## Activation
In order to activate API functionality on your Name.com account, you must apply to the API program. The application form is [located here](https://www.name.com/account/login). It usually takes a few days to get a response. After you are accepted, you should receive your API token via email.

## Tips and error messages

### invalid character '<'

```text
integration_test.go:140: api returned unexpected response: invalid character '<' looking for beginning of value
```

This error means an invalid URL is being used to reach the API endpoint.  It usually means a setting is `api.name.com/api` when `api.name.com` is correct (i.e. remove the `/api`).

In `creds.json`, remove the extra `/api` from `apiurl`, or omit `apiurl` to use the default.

TODO(tlim): Improve the error message. (Volunteer needed!)

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
    - Not supported: [PTR](../language-reference/domain-modifiers/PTR.md) (PTR records are not supported (See Link): https://www.name.com/support/articles/205188508-Reverse-DNS-records)
<!-- provider-features-end -->
