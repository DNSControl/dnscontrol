## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `GIGAHOST`
along with your [Gigahost](https://gigahost.no/) API key.

Example:

{% code title="creds.json" %}
```json
{
  "gigahost": {
    "TYPE": "GIGAHOST",
    "apikey": "flux_live_your-api-key"
  }
}
```
{% endcode %}

The [creds.json](../commands/creds-json.md#example-commands) page in the docs explains how you can generate this dynamically so you can pull the secret token from 1Password or the vault of your choosing.

## Metadata

This provider does not recognize any special metadata fields unique to Gigahost.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_GIGAHOST = NewDnsProvider("gigahost");

D("example.com", REG_NONE, DnsProvider(DSP_GIGAHOST),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

Gigahost is a DNS Service Provider only; it is not a registrar in DNSControl.
If your domain is registered with Gigahost, set the nameservers to Gigahost's
(see [Nameservers](#nameservers) below) at your registrar.

## Activation

1. Log in to the [Gigahost control panel](https://gigahost.no/).
2. Create an API key with the **DNS read-write** permission. Keys are prefixed
   with `flux_live_`.
3. Put the key in `creds.json` as the `apikey` field shown above.

## Supported record types

This provider supports the following record types:

| Name  | Description |
| ----- | ----------- |
| A     | IPv4 address record |
| AAAA  | IPv6 address record |
| ALIAS | CNAME-like apex alias record |
| CAA   | Certification Authority Authorization record |
| CNAME | Canonical name (alias) record |
| DNAME | Delegation name record |
| MX    | Mail exchange record |
| NAPTR | Naming Authority Pointer record |
| NS    | Name server record |
| PTR   | Pointer record |
| SRV   | Service record |
| TXT   | Text record |

Record types not in this list (for example `TLSA`, `SSHFP`, `HTTPS`, `SVCB`,
`DS`, `LOC`) are rejected by the Gigahost API and are not supported. Any
unsupported record type already present in a zone is left untouched: the
provider ignores it on read (emitting a warning) so it is neither modified nor
deleted.

## Nameservers

Gigahost serves every zone it hosts from a fixed set of nameservers:

- `ns1.gigahost.no`
- `ns2.gigahost.no`
- `ns3.gigahost.no`

The provider returns these via `GetNameservers`, so DNSControl will suggest the
correct delegation automatically. Set these nameservers at your registrar to
delegate a domain to Gigahost.

## Limitations

### Zone creation

Zones must already exist in your Gigahost account. The provider does not create
new zones; create them in the Gigahost control panel first.

### Zone apex SOA

The zone apex `SOA` record is managed by Gigahost and is not exposed for
editing. The provider ignores it.

### Concurrent operations

The provider does not support concurrent API operations. Changes are applied
sequentially.
