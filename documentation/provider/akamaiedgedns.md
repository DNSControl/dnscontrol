"Akamai Edge DNS Provider" configures Akamai's [Edge DNS](https://www.akamai.com/products/edge-dns) service.

This provider interacts with Edge DNS via the [Edge DNS Zone Management API](https://techdocs.akamai.com/edge-dns/reference/edge-dns-api).

Before you can use this provider, you need to create an "API Client" with authorization to use the [Edge DNS Zone Management API](https://techdocs.akamai.com/edge-dns/reference/edge-dns-api).

See the "Get Started" section of [Edge DNS Zone Management API](https://techdocs.akamai.com/edge-dns/reference/edge-dns-api), which says, "To enable this API, choose the API service named DNS—Zone Record Management, and set the access level to READ-WRITE."

Follow directions at [Authenticate With EdgeGrid](https://www.akamai.com/developer) to generate the required credentials.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `AKAMAIEDGEDNS` along with the authentication fields.

Example:

{% code title="creds.json" %}
```json
{
  "akamaiedgedns": {
    "TYPE": "AKAMAIEDGEDNS",
    "client_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "host": "akaa-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.xxxx.akamaiapis.net",
    "access_token": "akaa-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "client_token": "akaa-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "contract_id": "X-XXXX",
    "group_id": "NNNNNN"
  }
}
```
{% endcode %}

## Limitations

### Records

#### AKAMAICDN

The AKAMAICDN target must be an Edge Hostname preconfigured in your Akamai account.

The AKAMAICDN record must have a TTL of 20 seconds. Note that `dnscontrol preview` will not flag an incorrect TTL as an error; the TTL mismatches are only caught during `dnscontrol push`.

The AKAMAICDN record may only be used at the zone apex (`@`) if an AKAMAITLC record hasn't been used.

#### AKAMAITLC

The AKAMAITLC record can only be used at the zone apex (`@`).

The AKAMAITLC record can only be used once per zone.

#### ALIAS
Akamai Edge DNS does directly support `ALIAS` records. This provider will convert `ALIAS` records used at the zone apex (`@`) to `AKAMAITLC` records, and any other names to `CNAME` records.

### Secondary zones

This provider only supports creating primary zones in Akamai. If a secondary zone has been manually created, only `AKAMAICDN` and `AKAMAITLC` records can be managed, as all other records are read-only.

## Usage
Store your zone configuration details in a dnsconfig.js file in the same folder where the creds.json file is present.

Akamai assigns a unique set of authoritative nameservers for each contract.  These authorities should be used as the NS records on all zones belonging to this contract.

The NS records for these authorities have a TTL of 86400.

Add:

{% code title="dnsconfig.js" %}
```javascript
NAMESERVER_TTL(86400)
```
{% endcode %}

modifier to the dnscontrol.js D() function so that DNSControl does not change the TTL of the authoritative NS records.

Example `dnsconfig.js`:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_AKAMAIEDGEDNS = NewDnsProvider("akamaiedgedns");

D("example.com", REG_NONE, DnsProvider(DSP_AKAMAIEDGEDNS),
  NAMESERVER_TTL(86400),
  AUTODNSSEC_ON,
  AKAMAICDN("@", "www.preconfigured.edgesuite.net", TTL(20)),
  A("foo", "1.2.3.4"),
);
```
{% endcode %}

**Note:** A CNAME and an AKAMAICDN record with the same name is allowed.

**Note:** TTL for AKAMAICDN record must always be set to 20.

AKAMAICDN is a proprietary record type that is used to configure [Zone Apex Mapping](https://www.akamai.com/blog/security/edge-dns--zone-apex-mapping---dnssec). The AKAMAICDN target must be preconfigured in the Akamai network.

### dnscontrol check command
```shell
dnscontrol check
```

Use **dnscontrol check** to verify whether the dnsconfig.js file contents are valid.

Example:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_AKAMAIEDGEDNS = NewDnsProvider("akamaiedgedns");

D("example.com", REG_NONE, DnsProvider(DSP_AKAMAIEDGEDNS),
  NAMESERVER_TTL(86400),
  AUTODNSSEC_ON,
  AKAMAICDN("@", "www.preconfigured.edgesuite.net", TTL(20)),
  A("foo", "1.2.3.4"),
);
```
{% endcode %}

Output:
```
No errors.
```

### dnscontrol preview command
```shell
dnscontrol preview
```
Use `dnscontrol preview` to see which DNS changes would be made by `dnscontrol push`—without applying them.

Example:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_AKAMAIEDGEDNS = NewDnsProvider("akamaiedgedns");

D("example.com", REG_NONE, DnsProvider(DSP_AKAMAIEDGEDNS),
  NAMESERVER_TTL(86400),
  AUTODNSSEC_ON,
  AKAMAICDN("@", "www.preconfigured.edgesuite.net", TTL(20)),
  A("foo", "1.2.3.4"),
);
```
{% endcode %}

**Note:** If the zone does not exist `dnscontrol preview` returns an error:
```
******************** Domain: example.com
1 correction (akamaiedgedns)
#1: Ensuring zone "example.com" exists in "akamaiedgedns"
CONCURRENTLY gathering 0 zone(s)
SERIALLY gathering 1 zone(s)
Serially Gathering: "example.com"
******************** Domain: example.com
INFO#1: Domain "example.com" provider akamaiedgedns Error: recordset list retrieval failed. error: Title: Not Found; Type: https://problems.luna.akamaiapis.net/authoritative-dns/notFound; Detail: Unable to find zone 'example.com'
Done. 1 corrections.
completed with errors
```

**Note:** If the zone does not exist and you want to see the changes which will be made by dnscontrol push then use `dnscontrol preview` with the `--populate-on-preview` flag specified. This automatically creates the zone with SOA and NS records.

Command:
```
dnscontrol preview --populate-on-preview
```

Output:
```
******************** Domain: example.com
1 correction (akamaiedgedns)
#1: Ensuring zone "example.com" exists in "akamaiedgedns"
Created zone: example.com
  Type: PRIMARY
  Comment: This zone created by DNSControl (http://dnscontrol.org)
  SignAndServe: false
  SignAndServeAlgorithm: RSA_SHA512
  ContractId: X-XXXXXX
  GroupId: NNNNN
SUCCESS!
CONCURRENTLY gathering 0 zone(s)
SERIALLY gathering 1 zone(s)
Serially Gathering: "example.com"
******************** Domain: example.com
2 corrections (akamaiedgedns)
#1: + CREATE AKAMAICDN example.com www.preconfigured.edgesuite.net ttl=20
#2: + CREATE A foo.example.com 1.2.3.4 ttl=300
#3: Enable AutoDnsSec
```
In the above example since, the zone `example.com` did not exist, running `dnscontrol preview` with the `--populate-on-preview` flag created a zone named example.com with only the NS and SOA records and showed what changes will be applied by `dnscontrol push`.

### dnscontrol push command
```shell
dnscontrol push
```
Use `dnscontrol push` to create a new zone or update an existing zone.

#### Creating a New Zone

Example:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_AKAMAIEDGEDNS = NewDnsProvider("akamaiedgedns");

D("example_2.com", REG_NONE, DnsProvider(DSP_AKAMAIEDGEDNS),
  NAMESERVER_TTL(86400),
  AUTODNSSEC_ON,
  AKAMAICDN("@", "www.preconfigured_2.edgesuite.net", TTL(20)),
  A("foo_2", "1.2.3.5"),
);
```
{% endcode %}

Output:
```
******************** Domain: example_2.com
1 correction (akamaiedgedns)
#1: Ensuring zone "example_2.com" exists in "akamaiedgedns"
Created zone: example_2.com
  Type: PRIMARY
  Comment: This zone created by DNSControl (http://dnscontrol.org)
  SignAndServe: false
  SignAndServeAlgorithm: RSA_SHA512
  ContractId: X-XXXXXX
  GroupId: NNNNN
SUCCESS!
CONCURRENTLY gathering 0 zone(s)
SERIALLY gathering 1 zone(s)
Serially Gathering: "example_2.com"
******************** Domain: example_2.com
2 corrections (akamaiedgedns)
#1: + CREATE AKAMAICDN example_2.com www.preconfigured_2.edgesuite.net ttl=20
SUCCESS!
#2: + CREATE A foo_2.example_2.com 1.2.3.5 ttl=300
SUCCESS!
#3: Enable AutoDnsSec
```
In the above example since, zone `example_2.com` did not exist running `dnscontrol push` created a new zone `example_2.com` with NS, SOA and the other records (In this example, AKAMAICDN and A records).

#### Updating an Existing Zone

#### Important Note:
- When running the `dnscontrol push` command to update an existing DNS zone, the dnsconfig.js file must include all records for that zone—not just the ones being modified.
- Any records that exist in Akamai EdgeDNS but are not present in the dnsconfig.js file will be deleted during the push, as dnscontrol treats the file as the authoritative source.

Example 1

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_AKAMAIEDGEDNS = NewDnsProvider("akamaiedgedns");

D("example.com", REG_NONE, DnsProvider(DSP_AKAMAIEDGEDNS),
  NAMESERVER_TTL(86400),
  AUTODNSSEC_ON,
  AKAMAICDN("@", "www.preconfigured.edgesuite.net", TTL(20)),
  A("foo", "1.2.3.4"),
);
```
{% endcode %}

Output:
```
CONCURRENTLY gathering 0 zone(s)
SERIALLY gathering 1 zone(s)
Serially Gathering: "example.com"
******************** Domain: example.com
2 corrections (akamaiedgedns)
#1: + CREATE AKAMAICDN example.com www.preconfigured.edgesuite.net ttl=20
SUCCESS!
#2: + CREATE A foo.example.com 1.2.3.4 ttl=300
SUCCESS!
#3: Enable AutoDnsSec

SUCCESS!
Done. 2 corrections.
```
Since, the zone `example.com` was created with SOA and NS when the command `dnscontrol preview --populate-on-preview` ran, running `dnscontrol push` adds the AKAMAICDN and A records.

Example 2

In this example the A record is updated to have the IP **1.2.3.10** from **1.2.3.4**.

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_AKAMAIEDGEDNS = NewDnsProvider("akamaiedgedns");

D("example.com", REG_NONE, DnsProvider(DSP_AKAMAIEDGEDNS),
  NAMESERVER_TTL(86400),
  AUTODNSSEC_ON,
  AKAMAICDN("@", "www.preconfigured.edgesuite.net", TTL(20)),
  A("foo", "1.2.3.10"),
);
```
{% endcode %}

Output:
```
CONCURRENTLY gathering 0 zone(s)
SERIALLY gathering 1 zone(s)
Serially Gathering: "example.com"
******************** Domain: example.com
1 correction (akamaiedgedns)
#1: ± MODIFY A foo.example.com: (1.2.3.4 ttl=300) -> (1.2.3.10 ttl=300)
SUCCESS!
Done. 1 corrections.
```

### dnscontrol create-domains
```shell
dnscontrol create-domains
```
automatically creates SOA and authoritative NS records.

Example:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_AKAMAIEDGEDNS = NewDnsProvider("akamaiedgedns");

D("example_3.com", REG_NONE, DnsProvider(DSP_AKAMAIEDGEDNS),
  NAMESERVER_TTL(86400),
  AUTODNSSEC_ON,
  AKAMAICDN("@", "www.preconfigured_3.edgesuite.net", TTL(20)),
  A("foo_3", "1.2.3.6"),
);
```
{% endcode %}

Output:
```
DEPRECATED: This command is deprecated. The domain is automatically created at the Domain Service Provider during the push command.
DEPRECATED: To prevent disable auto-creating, use --no-populate with the push command.
***  example_3.com
  - akamaiedgedns
Created zone: example_3.com
  Type: PRIMARY
  Comment: This zone created by DNSControl (http://dnscontrol.org)
  SignAndServe: false
  SignAndServeAlgorithm: RSA_SHA512
  ContractId: X-XXXXXX
  GroupId: NNNNN
```

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ✅
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
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
