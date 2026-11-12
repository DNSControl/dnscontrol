## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `RWTH`
along with your API Token (which you can create via noc-portal.rz.rwth-aachen.de/dns-admin/en/api_tokens).

The provider may only be used from within the intranet.

Example:

{% code title="creds.json" %}
```json
{
  "rwth": {
    "TYPE": "RWTH",
    "api_token": "bQGz0DOi0AkTzG...="
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to it.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.rwth-aachen.de", REGISTRAR(), SERVICE("rwth"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Caveats
The default TTL is not automatically fetched, as the API does not provide such an endpoint.

The RWTH deploys zones every 15 minutes, so it might take some time for changes to take effect.

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TXT](../language-reference/domain-modifiers/TXT.md)
    - Unimplemented: [DS](../language-reference/domain-modifiers/DS.md) (DS records are only supported at the apex and require a different API call that hasn't been implemented yet.)
<!-- provider-features-end -->
