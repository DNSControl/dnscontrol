# MikroTik RouterOS DNS Provider

This DNS provider manages DNS static entries on a MikroTik RouterOS device via its REST API.

## Supported Features

- `dnscontrol get-zones` is supported. Zones are auto-detected by grouping record names by their effective domain suffix.
- Supported record types: `A`, `AAAA`, `CNAME`, `MX`, `NS`, `SRV`, `TXT`
- Custom record types:
  - [`MIKROTIK_FWD`](../language-reference/domain-modifiers/MIKROTIK_FWD.md) — RouterOS FWD entries for conditional DNS forwarding with address list population.
  - [`MIKROTIK_NXDOMAIN`](../language-reference/domain-modifiers/MIKROTIK_NXDOMAIN.md) — RouterOS NXDOMAIN entries (respond with NXDOMAIN for matching queries).
  - [`MIKROTIK_FORWARDER`](../language-reference/domain-modifiers/MIKROTIK_FORWARDER.md) — RouterOS DNS forwarder entries (managed via the synthetic `_forwarders.mikrotik` zone).

## Configuration

The provider is configured using entries in `creds.json`:

- `host`: RouterOS REST API endpoint (e.g. `http://192.168.88.1:8080`)
- `username`: RouterOS user with API access
- `password`: Password for the user
- `zonehints`: (optional) Comma-separated list of zone names to help identify zones with 3+ labels (e.g. `internal.corp.local,home.arpa,home.example.com`)

Example `creds.json` entry:

```json
{
  "mikrotik": {
    "TYPE": "MIKROTIK",
    "host": "http://192.168.88.1:8080",
    "username": "admin",
    "password": "secret",
    "zonehints": "internal.corp.local,home.arpa,home.example.com"
  }
}
```

### Zone Detection

RouterOS has no native zone concept — DNS static entries are a flat list. The provider groups records into zones by their domain suffix:

1. If `zonehints` is configured, records are matched against hints (longest match wins).
2. Otherwise, `publicsuffix.EffectiveTLDPlusOne` is used for public TLDs.
3. For private TLDs (e.g. `.local`), the last two labels are used as a fallback.

Use `zonehints` when you have zones with 3+ labels (e.g. `h.example.com` as a separate zone from `example.com`).

## Record Metadata

All record types support the following metadata keys:

| Key               | Type   | Description                                                        |
|-------------------|--------|--------------------------------------------------------------------|
| `match_subdomain` | string | Set to `"true"` to enable RouterOS subdomain matching.             |
| `regexp`          | string | RouterOS regexp for matching queries.                              |
| `address_list`    | string | RouterOS address list to populate with resolved addresses.         |
| `comment`         | string | Comment stored on the RouterOS record.                             |

### [`MIKROTIK_FWD`](../language-reference/domain-modifiers/MIKROTIK_FWD.md)

Forward DNS queries to a specified upstream server. The target can be an IP address or the name of a [`MIKROTIK_FORWARDER`](../language-reference/domain-modifiers/MIKROTIK_FORWARDER.md) entry. Commonly used for conditional forwarding with address list population.

```javascript
MIKROTIK_FWD("@", "8.8.8.8", {match_subdomain: "true", address_list: "vpn-list"})
MIKROTIK_FWD("@", "my-upstream", {match_subdomain: "true"})  // reference a forwarder by name
```

### [`MIKROTIK_NXDOMAIN`](../language-reference/domain-modifiers/MIKROTIK_NXDOMAIN.md)

Return NXDOMAIN for matching queries. Used for DNS-based blocking (e.g. ad blocking).

```javascript
MIKROTIK_NXDOMAIN("ads"),                                  // block ads.example.com
MIKROTIK_NXDOMAIN("tracking", {match_subdomain: "true"}),   // block tracking.example.com and all subdomains
```

### [`MIKROTIK_FORWARDER`](../language-reference/domain-modifiers/MIKROTIK_FORWARDER.md)

Manage RouterOS DNS forwarder entries via the synthetic `_forwarders.mikrotik` zone. The name can be a domain name or an arbitrary alias.

Additional metadata keys for forwarders:

| Key                | Type   | Description                                        |
|--------------------|--------|----------------------------------------------------|
| `doh_servers`      | string | DoH server URLs for this forwarder.                |
| `verify_doh_cert`  | string | Set to `"true"` to verify DoH certificate.         |
| `comment`          | string | Comment stored on the RouterOS forwarder entry.    |

```javascript
D("_forwarders.mikrotik", REG_CHANGEME,
  DnsProvider(DSP_MIKROTIK),
  MIKROTIK_FORWARDER("corp.example.com", "10.0.0.53,10.0.0.54"),
  MIKROTIK_FORWARDER("my-upstream", "1.1.1.1"),  // arbitrary alias
)
```

**Important:** `MIKROTIK_FWD` records can reference forwarder entries by name (e.g. `MIKROTIK_FWD("@", "my-upstream", ...)`). When using named forwarders, the `_forwarders.mikrotik` zone must appear **before** any zones that reference its entries in your `dnsconfig.js`. This ensures forwarder entries are created before the FWD records that depend on them. The `get-zones` command automatically outputs `_forwarders.mikrotik` first.

## Usage

```javascript
var DSP_MIKROTIK = NewDnsProvider("mikrotik", "MIKROTIK");
var REG_CHANGEME = NewRegistrar("none");

// Define forwarders first so they exist before being referenced.
D("_forwarders.mikrotik", REG_CHANGEME,
  DnsProvider(DSP_MIKROTIK),
  MIKROTIK_FORWARDER("corp.internal", "10.0.0.53,10.0.0.54"),
  MIKROTIK_FORWARDER("doh-upstream", "1.1.1.1", {doh_servers: "https://cloudflare-dns.com/dns-query", verify_doh_cert: "true"}),
)

D("example.com", REG_CHANGEME,
  {no_ns: "true"},
  DnsProvider(DSP_MIKROTIK),
  A("www", "192.0.2.1"),
  AAAA("www", "2001:db8::1"),
  CNAME("blog", "www.example.com."),
  MX("@", 10, "mail.example.com."),
  MIKROTIK_FWD("@", "doh-upstream", {match_subdomain: "true", address_list: "vpn-list"}),
  MIKROTIK_FWD("internal", "corp.internal", {match_subdomain: "true"}),
  MIKROTIK_NXDOMAIN("ads", {match_subdomain: "true"}),
)
```

**Note:** RouterOS does not expose nameservers, so `{no_ns: "true"}` should be set on all domains to suppress the "Skipping registrar" warning.

## Activation

The RouterOS REST API must be enabled on the device.

### Enable REST API (RouterOS 7.x)

Via the RouterOS CLI (SSH or terminal):

```
/ip/service/set www-ssl disabled=no
/certificate/add name=local-cert common-name=router
/ip/service/set www-ssl certificate=local-cert
```

Or for HTTP (not recommended for production):

```
/ip/service/set www disabled=no port=8080
```

### Create a dedicated user

```
/user/add name=dnscontrol password=secret group=full
```

For read-only preview, use `group=read`.

## Caveats

- **No native zone concept.** Zones are inferred from record names. Use `zonehints` for multi-label private zones.
- **Forwarder ordering.** If `MIKROTIK_FWD` records reference forwarder entries by name, the `_forwarders.mikrotik` zone must be defined before those zones in `dnsconfig.js`.
- **MX records with target `.` (null MX) are rejected** by the audit system.
- **Dynamic and disabled records are ignored** during zone enumeration and record fetching.
- **TTL values** are stored in RouterOS duration format (e.g. `1d`, `1h30m`) and converted automatically.

## Development Notes

This provider uses the RouterOS REST API endpoints:
- `/rest/ip/dns/static` — for DNS static records (A, AAAA, CNAME, FWD, MX, NS, NXDOMAIN, SRV, TXT)
- `/rest/ip/dns/forwarders` — for DNS forwarder entries

Records are compared using `diff2.ByRecord()` with custom comparison functions that include metadata fields (`match_subdomain`, `regexp`, `address_list`, `comment`) so that metadata-only changes are properly detected.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❌
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❌
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ❌
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❌
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
