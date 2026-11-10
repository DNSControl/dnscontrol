## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `OpenSRS`
along with your OpenSRS credentials.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE("opensrs"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ❌
  - Registrar: ✅
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
<!-- provider-features-end -->
