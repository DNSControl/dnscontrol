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
var REG_RTR = NewRegistrar("realtimeregister");
var DSP_RTR = NewDnsProvider("realtimeregister");

D("example.com", REG_RTR, DnsProvider(DSP_RTR),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ✅
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ✅
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❌
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
