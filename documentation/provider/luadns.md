## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `LUADNS`
along with your [email and API key](https://www.luadns.com/api.html#authentication).

Example:

{% code title="creds.json" %}
```json
{
  "luadns": {
    "TYPE": "LUADNS",
    "email": "your-email",
    "apikey": "your-api-key"
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to LuaDNS.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE("luadns"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Activation
[Create API key](https://api.luadns.com/api_keys).

## Caveats
- LuaDNS cannot change the default nameserver TTL in `nameserver_ttl`, it is forced to fixed at 86400("1d").
This is not the case if you are using vanity nameservers.
- This provider does not currently support the "FORWARD" and "REDIRECT" record types.
- The API is available on the LuaDNS free plan, but due to the limit of 30 records, some tests will fail when doing integration tests.
- Changes are sent in bulk requests rather than one request per record set. Only consecutive changes of the same kind (creates, updates or deletes) share a request, so the order DNSControl chose is kept; a request holds at most 100 records. If a request fails, all of the changes in it fail together.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
- Record types
  - [`A`](../language-reference/domain-modifiers/A.md) [`AAAA`](../language-reference/domain-modifiers/AAAA.md) [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md) [`CAA`](../language-reference/domain-modifiers/CAA.md) [`CNAME`](../language-reference/domain-modifiers/CNAME.md) [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md) [`MX`](../language-reference/domain-modifiers/MX.md) [`NS`](../language-reference/domain-modifiers/NS.md) [`OPENPGPKEY`](../language-reference/domain-modifiers/OPENPGPKEY.md) [`PTR`](../language-reference/domain-modifiers/PTR.md) [`SRV`](../language-reference/domain-modifiers/SRV.md) [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md) [`TLSA`](../language-reference/domain-modifiers/TLSA.md) [`TXT`](../language-reference/domain-modifiers/TXT.md)
<!-- provider-features-end -->
