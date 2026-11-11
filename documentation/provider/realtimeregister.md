[realtimeregister.com](https://realtimeregister.com) is a domain registrar based in the Netherlands.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `REALTIMEREGISTER`
along with your API-key. Further configuration includes a flag indicating BASIC or PREMIUM DNS-service and a flag
indicating the use of the sandbox environment

**Example:**

{% code title="creds.json" %}
```json
{
  "realtimeregister": {
    "TYPE": "REALTIMEREGISTER",
    "apikey": "abcdefghijklmnopqrstuvwxyz1234567890",
    "sandbox" : "0",
    "premium" : "0"
  }
}
```
{% endcode %}

If sandbox is omitted or set to any other value than "1" the production API will be used.
If premium is set to "1", you will only be able to update zones using Premium DNS. If it is omitted or set to any other value, you
will only be able to update zones using Basic DNS.

**Important Notes**:
* It is recommended to create a 'DNSControl' user in your account settings with limited permissions
(i.e. VIEW_DNS_ZONE, CREATE_DNS_ZONE, UPDATE_DNS_ZONE, VIEW_DOMAIN, UPDATE_DOMAIN), otherwise anyone with
access to this `creds.json` file might have *full* access to your RTR account and will be able to transfer or delete your domains.

## Metadata
This provider does not recognize any special metadata fields unique to Realtime Register.

## Usage
An example `dnsconfig.js` configuration file

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("realtimeregister"), SERVICE("realtimeregister"),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Feature Summary

<!-- provider-features-start -->
- Support Level and Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- API-level Features
  - [Concurrent fetching](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
  - [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
- Supported Record Types
  - [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
    - Not supported: [DS](../language-reference/domain-modifiers/DS.md) (Only for subdomains)
<!-- provider-features-end -->
