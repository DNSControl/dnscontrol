This provider uses the native DNS protocols. It uses the AXFR (RFC5936, Zone Transfer Protocol) protocol to retrieve existing records and uses DDNS (RFC2136, Dynamic Update) to make updates. It can use both TSIG (RFC2845) and IP-based authentication (ACLs).

It can work with any standards-compliant authoritative DNS server. It has been tested with [BIND](https://www.isc.org/bind/), [Knot](https://www.knot-dns.cz/), and [Yadifa](https://yadifa.eu/home.html).

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `AXFRDDNS`.

### Connection modes

Zone transfers and DDNS updates default to TCP when using this provider.

The following two parameters in `creds.json` allow switching to TCP or TCP over TLS.

* `update-mode`: May contain `tcp` (the default), `udp`, or `tcp-tls`.
* `transfer-mode`: May contain `tcp` (the default), or `tcp-tls`.

### Authentication

Authentication information is included in the `creds.json` entry for the provider:

* `transfer-key`: If this exists, the value is used to authenticate AXFR transfers.
* `update-key`: If this exists, the value is used to authenticate DDNS updates.

For instance, your `creds.json` might look like:

{% code title="creds.json" %}
```json
{
  "axfrddns": {
    "TYPE": "AXFRDDNS",
    "transfer-key": "hmac-sha256:transfer-key-id:Base64EncodedSecret=",
    "update-key": "hmac-sha256:update-key-id:AnotherSecret="
  }
}
```
{% endcode %}

If either key is missing, DNSControl defaults to IP-based ACL authentication for that function. Including both keys is the most secure option. Omitting both keys defaults to IP-based ACLs for all operations, which is the least secure option.

If distinct zones require distinct keys, you will need to instantiate the provider once for each key:

{% code title="dnsconfig.js" %}
```javascript
var DSP_AXFRDDNS_A = NewDnsProvider("axfrddns-a");
var DSP_AXFRDDNS_B = NewDnsProvider("axfrddns-b");
```
{% endcode %}

And update `creds.json` accordingly:

{% code title="creds.json" %}
```json
{
  "axfrddns-a": {
    "TYPE": "AXFRDDNS",
    "transfer-key": "hmac-sha256:transfer-key-id:Base64EncodedSecret=",
    "update-key": "hmac-sha256:update-key-id:AnotherSecret="
  },
  "axfrddns-b": {
    "TYPE": "AXFRDDNS",
    "transfer-key": "hmac-sha512:transfer-key-id-B:SmallSecret=",
    "update-key": "hmac-sha512:update-key-id-B:YetAnotherSecret="
  }
}
```
{% endcode %}

### Default nameservers

The AXFR+DDNS provider can be configured with a list of default nameservers. They will be added to all the zones handled by the provider.

This list can be provided either as metadata or in `creds.json`. Only the later allows `get-zones` to work properly.

{% code title="dnsconfig.js" %}
```javascript
var DSP_AXFRDDNS = NewDnsProvider("axfrddns", {
        "default_ns": [
            "ns1.example.com.",
            "ns2.example.com.",
            "ns3.example.com.",
            "ns4.example.com."
        ]
    }
)
```
{% endcode %}

{% code title="creds.json" %}
```json
{
  "axfrddns": {
    "TYPE": "AXFRDDNS",
    "nameservers": "ns1.example.com,ns2.example.com,ns3.example.com,ns4.example.com"
  }
}
```
{% endcode %}

### Primary master

By default, the AXFR+DDNS provider will send the AXFR requests and the DDNS updates to the first nameserver of the zone, usually known as the "primary master". Typically, this is the first of the default nameservers. Though, on some networks, the primary master is a private node, hidden behind slaves, and it does not appear in the `NS` records of the zone. In that case, the IP or the name of the primary server must be provided in `creds.json`. With this option, a non-standard port might be used.

{% code title="creds.json" %}
```json
{
  "axfrddns": {
    "TYPE": "AXFRDDNS",
    "master": "10.20.30.40:5353"
  }
}
```
{% endcode %}

When no nameserver appears in the zone, and no default nameservers nor custom master are configured, the AXFR+DDNS provider will fail with the following error message:

```text
[Error] AXFRDDNS: the nameservers list cannot be empty.
Please consider adding default `nameservers` or an explicit `master` in `creds.json`.
```

### Transfer/AXFR server

As mentioned above, the AXFR+DDNS provider will send AXFR requests to the primary master for the zone. On some networks, AXFR requests are handled by a different server than DDNS requests. Use the `transfer-server` option in `creds.json`. If not specified, it falls back to the primary master.

{% code title="creds.json" %}
```json
{
  "axfrddns": {
    "TYPE": "AXFRDDNS",
    "transfer-server": "233.252.0.0"
  }
}
```
{% endcode %}

### Example: local testing

When testing `dnscontrol` against a local nameserver, you might use the following minimal configuration:

{% code title="creds.json" %}
```json
{
  "axfrddns": {
    "TYPE": "AXFRDDNS",
    "master": "127.0.0.1"
  }
}
```
{% endcode %}

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DNS = NewDnsProvider("axfrddns", {
    default_ns: [
        "ns.example.com.",
    ],
});

D("example.com", REG_NONE, DnsProvider(DNS),
    A("ns", "127.0.0.1")
)
```
{% endcode %}

## Server configuration examples

### Bind9

This is a sample `named.conf` example for an authoritative server hosting the zone `example.com`. It uses a simple IP-based ACL for the AXFR transfer and a conjunction of TSIG and IP-based ACL for the updates.

{% code title="named.conf" %}
```text
options {

    listen-on { any; };
    listen-on-v6 { any; };

    allow-query { any; };
    allow-notify { none; };
    allow-recursion { none; };
    allow-transfer { none; };
    allow-update { none; };
    allow-query-cache { none; };

};

zone "example.com" {
  type master;
  file "/etc/bind/db.example.com";
  allow-transfer { example-transfer; };
  allow-update { example-update; };
};

## Allow transfer to anyone on our private network

acl example-transfer {
    172.17.0.0/16;
};

## Allow update only from authenticated client on our private network

acl example-update {
  ! {
   !172.17.0.0/16;
   any;
  };
  key update-key-id;
};

key update-key-id {
  algorithm HMAC-SHA256;
  secret "AnotherSecret=";
};
```
{% endcode %}

## FYI: get-zones

When using `get-zones`, a custom master or a list of default nameservers should be configured in `creds.json`.

The AXFR+DDNS provider does not display DNSSEC records. But, if any DNSSEC records are found in the zone, it will replace all of them with a single placeholder record:

```text
    __dnssec         IN TXT   "Domain has DNSSec records, not displayed here."
```

## FYI: create-domain

The AXFR+DDNS provider is not able to create domain.

## FYI: AUTODNSSEC

The AXFR+DDNS provider is not able to ask the DNS server to sign the zone. But, it is able to check whether the server seems to do so or not.

When AutoDNSSEC is enabled, the AXFR+DDNS provider will emit a warning when no RRSIG, DNSKEY or NSEC records are found in the zone.

When AutoDNSSEC is disabled, the AXFR+DDNS provider will emit a warning when RRSIG, DNSKEY or NSEC records are found in the zone.

When AutoDNSSEC is not enabled or disabled, no checking is done.

## FYI: MD5 Support

By default the used DNS Go package by miekg has deprecated support for the (insecure) MD5 algorithm [https://github.com/miekg/dns/commit/93945c284489394b77653323d11d5de83a2a6fb5](https://github.com/miekg/dns/commit/93945c284489394b77653323d11d5de83a2a6fb5). Some providers like the Leibniz Supercomputing Centre (LRZ) located in Munich still use this algorithm to authenticate internal dynamic DNS updates. To compensate for the lack of MD5, a custom MD5 TSIG Provider was added into DNSControl.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ❌
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ✅
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ✅
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
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
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ✅
<!-- provider-features-end -->
