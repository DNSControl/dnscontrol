---
name: IGNORE_EXTERNAL_DNS
parameters:
    - prefix
parameter_types:
    prefix: string?
---

`IGNORE_EXTERNAL_DNS` makes DNSControl automatically detect and ignore DNS records managed by Kubernetes external-dns.

## Background

[External-dns](https://github.com/kubernetes-sigs/external-dns) is a popular Kubernetes controller that synchronizes exposed Kubernetes Services and Ingresses with DNS providers. It creates DNS records automatically based on annotations on your Kubernetes resources.

External-dns uses TXT records to track ownership of the DNS records it manages. These TXT records contain metadata in this format:

```
"heritage=external-dns,external-dns/owner=<owner-id>,external-dns/resource=<resource>"
```

When you have both DNSControl and external-dns managing the same DNS zone, conflicts can occur. DNSControl will try to delete records created by external-dns, and external-dns will recreate them, leading to an endless update cycle.

## How it works

When `IGNORE_EXTERNAL_DNS` is enabled, DNSControl will:

1. Scan existing TXT records for the external-dns heritage marker (`heritage=external-dns`)
2. Parse the TXT record name to determine which DNS record it manages
3. Automatically ignore both the TXT ownership record and the corresponding DNS record

External-dns creates TXT records with prefixes based on record type:
- `a-<name>` for A records
- `aaaa-<name>` for AAAA records
- `cname-<name>` for CNAME records
- `ns-<name>` for NS records
- `mx-<name>` for MX records
- `srv-<name>` for SRV records
- `txt-<name>` for TXT records (when external-dns manages TXT records)

For example, if external-dns creates an A record at `myapp.example.com`, it will also create a TXT record at `a-myapp.example.com` containing the heritage information.

## Usage

{% code title="dnsconfig.js" %}
```javascript
// Default: detect standard external-dns prefixes (a-, cname-, etc.)
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  IGNORE_EXTERNAL_DNS(),
  // Your static DNS records managed by DNSControl
  A("www", "1.2.3.4"),
  A("mail", "1.2.3.5"),
  MX("@", 10, "mail"),
  // Records created by external-dns (from Kubernetes Ingresses/Services)
  // will be automatically detected and ignored
);
```
{% endcode %}

## Custom Prefix Support

If your external-dns is configured with a custom `--txt-prefix` (as documented in the [external-dns TXT registry docs](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/registry/txt.md#prefixes-and-suffixes)), pass that prefix to `IGNORE_EXTERNAL_DNS()`:

{% code title="dnsconfig.js" %}
```javascript
// If external-dns is configured with --txt-prefix="extdns-"
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  IGNORE_EXTERNAL_DNS("extdns-"),
  A("www", "1.2.3.4"),
);
```
{% endcode %}

This will match TXT records like `extdns-www`, `extdns-api`, etc.

Without a prefix argument, it detects:
- The default `%{record_type}-` format (prefixes like `a-`, `cname-`, etc.)
- Legacy format (TXT record with same name as managed record)

## Example scenario

Suppose you have:
- A Kubernetes cluster running external-dns with `--txt-owner-id=my-cluster`
- An Ingress resource that creates an A record for `myapp.example.com` pointing to `10.0.0.1`

External-dns will create:
1. An A record: `myapp.example.com` → `10.0.0.1`
2. A TXT record: `a-myapp.example.com` → `"heritage=external-dns,external-dns/owner=my-cluster,external-dns/resource=ingress/default/myapp"`

With `IGNORE_EXTERNAL_DNS` enabled, DNSControl will:
- Detect the TXT record at `a-myapp.example.com` as an external-dns ownership record
- Ignore both the TXT record and the A record at `myapp.example.com`
- Only manage the records you explicitly define in your `dnsconfig.js`

## Comparison with other options

| Feature | Use case |
|---------|----------|
| `IGNORE_EXTERNAL_DNS` | Automatically ignore all external-dns managed records |
| `IGNORE("*.k8s", "A,AAAA,CNAME,TXT")` | Ignore records under a specific subdomain pattern |
| `NO_PURGE` | Don't delete any records (less precise, records may accumulate) |

## Caveats

### One per domain

Only one `IGNORE_EXTERNAL_DNS()` should be used per domain. If you call it multiple times, the last prefix wins. If you have multiple external-dns instances with different prefixes managing the same zone, use `IGNORE()` patterns for additional prefixes.

### TXT Registry Format

This feature relies on external-dns's [TXT registry](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/registry/txt.md), which is the default registry type. The TXT record content format is well-documented:

```
"heritage=external-dns,external-dns/owner=<owner-id>,external-dns/resource=<resource>"
```

This feature detects the `heritage=external-dns` marker in TXT records to identify external-dns managed records.

### Custom Prefix Support

This feature supports custom prefixes configured via external-dns's `--txt-prefix` flag. If you're using a custom prefix, pass it to `IGNORE_EXTERNAL_DNS()`:

```javascript
// If external-dns uses --txt-prefix="extdns-"
IGNORE_EXTERNAL_DNS("extdns-")

// If external-dns uses --txt-prefix="myprefix-%{record_type}-"
IGNORE_EXTERNAL_DNS("myprefix-")  // The record type part is handled automatically

// If external-dns uses --txt-prefix="extdns-%{record_type}." (period format)
// This is recommended for apex domain support per external-dns docs
IGNORE_EXTERNAL_DNS("extdns-")  // Works with both hyphen and period format
```

Without a prefix argument, it detects:
- Default format: `%{record_type}-` prefix (e.g., `a-`, `cname-`)
- Legacy format: Same name as managed record (no prefix)

#### Period Format for Apex Domains

If you need external-dns to manage apex (root) domain records, the external-dns documentation recommends using a prefix with `%{record_type}` followed by a period:

```yaml
# external-dns deployment args
args:
  - --txt-prefix=extdns-%{record_type}.
```

This creates TXT records like `extdns-a.www` for the `www` A record, and `extdns-a` for the apex A record. DNSControl's `IGNORE_EXTERNAL_DNS` supports both formats:

- Hyphen format: `extdns-a-www` (from `--txt-prefix=extdns-` with default `%{record_type}-`)
- Period format: `extdns-a.www` (from `--txt-prefix=extdns-%{record_type}.`)

**Note:** Suffix-based naming (`--txt-suffix`) is not currently supported.

### Unsupported Registries

External-dns supports multiple registry types. This feature **only** supports:

- ✅ **TXT registry** (default) - Stores metadata in TXT records

The following registries are **not supported**:

- ❌ **DynamoDB registry** - Stores metadata in AWS DynamoDB
- ❌ **AWS-SD registry** - Stores metadata in AWS Service Discovery
- ❌ **noop registry** - No metadata persistence

### Legacy TXT Format

External-dns versions prior to v0.16 created TXT records without the record type prefix (e.g., `myapp.example.com` instead of `a-myapp.example.com`). This legacy format is supported but may match more records than intended since the record type cannot be determined.

## See also

* [`IGNORE`](IGNORE.md) for manually ignoring specific records with glob patterns
* [`NO_PURGE`](NO_PURGE.md) for preventing deletion of all unmanaged records
* [External-dns documentation](https://github.com/kubernetes-sigs/external-dns)
