## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `LINODE`
along with your [Linode Personal Access Token](https://cloud.linode.com/profile/tokens).

Example:

{% code title="creds.json" %}
```json
{
  "linode": {
    "TYPE": "LINODE",
    "token": "your-linode-personal-access-token"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Linode.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE("linode"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation

[Create Personal Access Token](https://cloud.linode.com/profile/tokens)

## Caveats

Linode does not allow all TTLs, but only a specific subset of TTLs. The following TTLs are supported
([source](https://www.linode.com/docs/api/domains/#domains-list__responses)):

- 0 (Default: the _Default TTL_ configured for the zone)
- 30 ("30s")
- 120 ("2m")
- 300 ("5m")
- 3600 ("1h")
- 7200 ("2h")
- 14400 ("4h")
- 28800 ("8h")
- 57600 ("16h")
- 86400 ("1d")
- 172800 ("2d")
- 345600 ("4d")
- 604800 ("1w")
- 1209600 ("2w")
- 2419200 ("4w")

{% hint style="info" %}
The _default TTL_ for a zone may be set using Linode's web UI or API. This sets the TTL for the SOA, NS records, and any other records that have their TTL set to zero. It also sets the value in the _minimum TTL_ field of the SOA. There currently seems to be no way to set the minimum TTL to a different value than the _default TTL_.
{% endhint %}

The provider will automatically round up your TTL to one of these values. For example, 600 seconds would become 3600
seconds, but 300 seconds would stay 300 seconds.

Linode validates labels of [`SRV`](../language-reference/domain-modifiers/SRV.md)
records more strictly than DNSControl, only permitting certain service names
and protocols.  Most issues will be caught at `check` and `preview` but certain
errors will not be caught until DNSControl `push` as they rely on the API
rejecting the update.  Also be aware that some invalid service names are
silently corrected by Linode, in which case every `push` will try to undo the
correction. All of these caveats only affect invalid inputs.

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
  - [get-zones](../commands/get-zones.md): ❌
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
