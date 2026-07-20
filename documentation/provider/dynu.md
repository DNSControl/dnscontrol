## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `DYNU`
along with your Dynu API key. You can generate an API key from the
[Dynu Control Panel](https://www.dynu.com/en-US/ControlPanel) under **API Credentials**.

Example:

{% code title="creds.json" %}
```json
{
  "dynu": {
    "TYPE": "DYNU",
    "api_key": "your-dynu-api-key"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Dynu.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_DYNU = NewDnsProvider("dynu");

D("example.com", REG_NONE, DnsProvider(DSP_DYNU),
    A("test", "1.2.3.4"),
    MX("@", 10, "mail.example.com."),
    TXT("@", "v=spf1 include:example.com ~all"),
);
```
{% endcode %}

## Activation

1. Log in to the [Dynu Control Panel](https://www.dynu.com/en-US/ControlPanel).
2. Navigate to **API Credentials**.
3. Generate a new API key.
4. Add the key to `creds.json` as shown above.

## Supported record types

Dynu supports the following DNS record types. The **Provider** column indicates
whether the DNSControl Dynu provider currently implements that type.

| Type       | Description                          | Provider  | Notes |
| ---------- | ------------------------------------ | :-------: | ----- |
| A          | IPv4 address                         | ✅        | |
| AAAA       | IPv6 address                         | ✅        | |
| AFSDB      | AFS Database                         | ✅        | |
| CAA        | Certification Authority Authorization| ✅        | |
| CERT       | Certificate                          | ✅        | |
| CNAME      | Canonical Name                       | ✅        | |
| DHCID      | DHCP Identifier                      | ✅        | |
| DNAME      | Delegation Name                      | ✅        | |
| HINFO      | System Information                   | ✅        | |
| HTTPS      | HTTPS Service Binding                | ✅        | |
| KEY        | Public Key                           | ✅        | |
| LOC        | Location Information                 | ✅        | |
| MX         | Mail Exchange                        | ✅        | |
| NAPTR      | Name Authority Pointer               | ✅        | |
| OPENPGPKEY | OpenPGP Key                          | ✅        | |
| PTR        | Pointer                              | ✅        | |
| RP         | Responsible Person                   | ✅        | |
| SMIMEA     | S/MIME Certificate Association       | ✅        | |
| SPF        | Sender Policy Framework              | ✅        | Normalised to TXT on read |
| SRV        | Service                              | ✅        | |
| SSHFP      | Secure Shell Fingerprint             | ✅        | |
| SVCB       | Service Binding                      | ✅        | |
| TLSA       | Transport Level Security             | ✅        | |
| TXT        | Text                                 | ✅        | Empty TXT not supported |
| URI        | Uniform Resource Identifier          | ✅        | |

## Caveats

### Apex NS records

Dynu manages its own authoritative nameservers (`ns1.dynu.com` through `ns6.dynu.com`) and does not permit creating, modifying, or deleting apex NS records via the API. DNSControl will not attempt to manage them. Subdomain NS delegations are fully supported.

### NS record TTL

Dynu forces all NS records to a TTL of 3600, regardless of the value specified in `dnsconfig.js`. TTL-only changes to NS records are silently ignored to maintain idempotency.

### SPF records

Dynu stores SPF records as a distinct record type internally, but the DNSControl provider normalises them to `TXT` on read. Write them as `TXT` records in `dnsconfig.js`.

### Wildcard records

Dynu does not support wildcard DNS records (e.g. `*.example.com`) via the API. DNSControl will reject them at audit time.

### SOA records

Dynu manages SOA records internally. They are not returned by the API and cannot be modified via DNSControl.

### Empty TXT records

Dynu rejects TXT records with an empty string value. DNSControl will reject them at audit time.

### Null MX targets

MX records with a null target (RFC 7505, priority 0, target `.`) are fully supported including direct updates.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❌
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ✅
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ✅
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ✅
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ✅
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
