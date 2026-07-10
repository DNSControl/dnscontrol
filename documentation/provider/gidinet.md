## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `GIDINET`
along with your Gidinet account credentials.

Example:

{% code title="creds.json" %}
```json
{
  "gidinet": {
    "TYPE": "GIDINET",
    "username": "your-gidinet-username",
    "password": "your-gidinet-password"
  }
}
```
{% endcode %}

The [creds.json](../commands/creds-json.md#example-commands) page in the docs explains how you can generate this dynamically so you can pull the secret token from 1Password or the vault of your choosing.

## Metadata

This provider does not recognize any special metadata fields unique to Gidinet.

## Usage

### As DNS Provider only

If you manage your domain registration elsewhere but want to use Gidinet for DNS:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_GIDINET = NewDnsProvider("gidinet");

D("example.com", REG_NONE, DnsProvider(DSP_GIDINET),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

### As both Registrar and DNS Provider

If your domain is registered with Gidinet and you want to manage both nameserver delegation and DNS records:

{% code title="dnsconfig.js" %}
```javascript
var REG_GIDINET = NewRegistrar("gidinet");
var DSP_GIDINET = NewDnsProvider("gidinet");

D("example.com", REG_GIDINET, DnsProvider(DSP_GIDINET),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

### As Registrar only (DNS hosted elsewhere)

If your domain is registered with Gidinet but you want to use a different DNS provider:

{% code title="dnsconfig.js" %}
```javascript
var REG_GIDINET = NewRegistrar("gidinet");
var DSP_OTHER = NewDnsProvider("cloudflare");

D("example.com", REG_GIDINET, DnsProvider(DSP_OTHER),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

When used as a registrar, Gidinet will manage the nameserver delegation at the registry level.

## Activation

1. Log in to the [Gidinet Control Panel](https://www.gidinet.com/)
2. Generate an **API password** at [https://www.gidinet.com/modules/private/account_password/](https://www.gidinet.com/modules/private/account_password/)
3. In `creds.json`, use your account **username** together with that **API password** -- the API password is distinct from your normal login password

## Supported record types

The Gidinet DNS API supports the following record types:

| Name  | Description |
| ----- | ----------- |
| A     | IPv4 address record |
| AAAA  | IPv6 address record |
| CNAME | Canonical name (alias) record |
| MX    | Mail exchange record |
| NS    | Name server record (subdomains only, apex NS managed by registrar) |
| TXT   | Text record |
| SRV   | Service record |

## Unsupported record types

The following record types are **not supported** by Gidinet:

- `ALIAS` - Not available
- `CAA` - Only available with premium service
- `DHCID`, `DNAME`, `DNSKEY`, `DS`, `HTTPS`, `LOC`, `NAPTR`, `PTR`, `SOA`, `SSHFP`, `SVCB`, `TLSA` - Not available

## Limitations

### TTL values

Gidinet only supports specific TTL values. If you specify a TTL that is not in the allowed list, DNSControl will automatically round up to the nearest allowed value.

Allowed TTL values (in seconds):
- 60 (1 minute)
- 300 (5 minutes)
- 600 (10 minutes)
- 900 (15 minutes)
- 1800 (30 minutes)
- 2700 (45 minutes)
- 3600 (1 hour)
- 7200 (2 hours)
- 14400 (4 hours)
- 28800 (8 hours)
- 43200 (12 hours)
- 64800 (18 hours)
- 86400 (1 day)
- 172800 (2 days)

### Nameservers

Gidinet offers two DNS tiers with different nameserver sets.

**Free tier (default):**
- `dnsl1.gidinet.com`
- `dnsl2.gidinet.com`

**Premium DNS:**
- `dns1.gidinet.com`
- `dns2.gidinet.com`
- `dns3.gidinet.com`
- `dns4.gidinet.com`
- `dns5.gidinet.com`

The DNS provider returns the free-tier nameservers via `GetNameservers`, so free-tier zones need no explicit `NAMESERVER(...)` — DNSControl will suggest the correct delegation to the registrar automatically:

{% code title="dnsconfig.js" %}
```javascript
var REG_GIDINET = NewRegistrar("gidinet");
var DSP_GIDINET = NewDnsProvider("gidinet");

D("example.com", REG_GIDINET, DnsProvider(DSP_GIDINET),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

For zones on the **premium DNS** tier, opt out of the free-tier defaults with `DnsProvider(DSP_GIDINET, 0)` and use the `GIDINET_PREMIUM_NS()` helper to emit the five premium `NAMESERVER()` records:

{% code title="dnsconfig.js" %}
```javascript
var REG_GIDINET = NewRegistrar("gidinet");
var DSP_GIDINET = NewDnsProvider("gidinet");

D("premium.example", REG_GIDINET,
    DnsProvider(DSP_GIDINET, 0),
    GIDINET_PREMIUM_NS(),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

The `0` passed to `DnsProvider()` tells DNSControl to skip the provider's auto-injected nameservers for that zone, so only the explicit `NAMESERVER()` records drive the delegation.

When used as a registrar, Gidinet updates the nameservers at the registry level via the Core API's `domainNameServersChange` method.

**Apex NS records are automatically filtered** by the DNS provider with a warning message. Gidinet does not support modifying NS records at the zone apex via the DNS API — they are managed by the registrar. If you use a DNS provider other than Gidinet, declare `NAMESERVER(...)` records (or rely on the other provider's `GetNameservers`) so `REG_GIDINET` can drive the delegation.

### Zone creation

Zones must be created via the Gidinet web interface. The API does not support creating new zones.

### Zone listing

The provider supports listing all zones in your account via `dnscontrol get-zones`. This uses the Core API's `domainGetList` method to retrieve all active domains.

### Concurrent operations

The provider does not support concurrent API operations. Changes are applied sequentially to ensure reliability.
