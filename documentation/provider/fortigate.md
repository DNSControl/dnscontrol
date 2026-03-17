# FortiGate DNS Provider

This DNS provider lets you manage DNS zones hosted on a Fortinet FortiGate device via its REST API.


## Supported Features

- `dnscontrol get-zones` is supported. Lists all DNS zones configured on the FortiGate device.
- Supported record types: `A`, `AAAA`, `CNAME`, `NS`, `MX`

## Configuration

The provider is configured using the following environment variables or entries in `creds.json`:

- `FORTIGATE_HOST`: The FortiGate host or IP address (e.g. `https://192.168.1.1`)
- `FORTIGATE_TOKEN`: API token with appropriate DNS permissions
- `FORTIGATE_VDOM`: (optional) Specify the virtual domain (default: `root`)
- `FORTIGATE_INSECURE_TLS`: (optional) Set to `true` to disable SSL certificate verification (useful for self-signed certs)
- `FORTIGATE_DEBUG_HTTP`: (optional) Set to `true` to log raw HTTP requests/responses

Example `creds.json` entry:

```json
{
  "FORTIGATE": {
    "host": "https://192.168.1.1",
    "token": "your-api-token",
    "vdom": "root",
    "insecure_tls": true,
    "debug_http": true
  }
}
```

## Metadata

### Domain Metadata

The following domain-level metadata keys are supported. These affect zone-level properties:

| Key            | Type    | Description                                                                 |
|----------------|---------|-----------------------------------------------------------------------------|
| `authoritative`| string  | Set to `"false"` to disable authoritative mode. Defaults to `"true"`.      |
| `forwarder`    | string  | Optional. IPv4 address to set as a forwarder for the zone.                  |

#### Example:

```javascript
D("example.com", REG_NONE, DnsProvider("FORTIGATE"),
  A("@", "192.0.2.1"),
  {
    metadata: {
      authoritative: "false",
      forwarder: "8.8.8.8"
    }
  }
)
```

If `forwarder` is provided, it must be a valid IPv4 address. An error will be raised otherwise.

### Record Metadata

| Key               | Type   | Applies to | Description                                       |
|-------------------|--------|------------|---------------------------------------------------|
| `fortigate_status`| string | All records | Set to `"disable"` to mark a record as disabled.  |

#### Example:

```javascript
A("test", "192.0.2.123", { metadata: { fortigate_status: "disable" } })
```

## Usage

To use this provider in a `dnsconfig.js`:

```javascript
D("example.com", REG_NONE, DnsProvider("FORTIGATE"),
  A("www", "192.0.2.1"),
  CNAME("blog", "external.example.net."),
  MX("@", 10, "mail.example.com."),
  NS("@", "ns1.example.net.")
)
```

## Activation

To enable DNS API access for DNSControl, a FortiGate admin user with an appropriate access profile must be created. This user should have `read-write` access to the system group to manage DNS zones and records.

### Step-by-step (via CLI)

Log in to the FortiGate CLI (via SSH or console), then run:

```bash
config global
config system accprofile
edit "DNSControl"
    set sysgrp read-write
next
end
```

This creates an admin profile named DNSControl with sufficient permissions to read and write DNS configuration.

### Assigning the profile to an API user

After creating the profile, create an admin user via CLI or GUI and assign the DNSControl profile to it. Then, generate an API token for this user.
Refer to Fortinet’s documentation on [How to configure REST API access](https://docs.fortinet.com/document/fortigate/7.6.3/administration-guide/399023/rest-api-administrator) for more details.
Once you have the token, use it in your `creds.json` as shown above.

## Caveats

- ✅ **NS and MX records are supported, with limitations:**  
  - Only apex records (hostname `"@"`) are supported.  
  - MX records must have a valid hostname (not `"."`).  
  - FortiGate does not enforce priority uniqueness or ordering.

- ❌ **PTR records are not supported.**  
  FortiGate stores reverse DNS data unconventionally. PTR records are excluded to prevent inconsistencies.

- ❌ **TXT records are not supported.**  
  The FortiGate API does not currently allow TXT records.

- ❌ **Wildcard records (`*`) are not supported.**  
  The FortiGate DNS engine does not support wildcard entries.


## Development notes

This provider uses the FortiGate REST API (`/api/v2/cmdb/system/dns-database`) to manage zones and DNS entries. It operates on the **"shadow" DNS database**, assuming zones are configured in **primary mode** (not forwarded). It automatically creates zones if they do not exist.

Debug logging of HTTP traffic can be enabled with the `debug_http` flag.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ✅
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❔
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ❔
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ❔
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❔
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❔
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
