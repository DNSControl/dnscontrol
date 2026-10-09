# Nameservers and Delegations

- [Credential entries](#credential-entries)
- [Typical Delegations](#typical-delegations)
  - [Same provider for REG and DNS](#same-provider-for-reg-and-dns)
  - [Different provider for REG and DNS](#different-provider-for-reg-and-dns)
  - [Registrar is elsewhere](#registrar-is-elsewhere)
  - [Domain is "nowhere"](#domain-is-nowhere)
  - [Zone is elsewhere](#zone-is-elsewhere)
  - [Override nameservers](#override-nameservers)
  - [Add nameservers](#add-nameservers)
  - [Shadow nameservers](#shadow-nameservers)
  - [Dual DNS Providers](#dual-dns-providers)
- [Other uses](#other-uses)
  - [Make zonefile backups](#make-zonefile-backups)
  - [Monitor delegation](#monitor-delegation)
- [Helper macros](#helper-macros)
  - [`DOMAIN_ELSEWHERE`](#domain_elsewhere)
  - [`DOMAIN_ELSEWHERE_AUTO`](#domain_elsewhere_auto)
- [Limits](#limits)


DNSControl can handle a variety of provider scenarios. The registrar and DNS provider can be the same company, different company, they can even be unknown! The document shows examples of many common and uncommon configurations.

## Credential entries

The names in `REGISTRAR()` and `SERVICE()` are example keys in `creds.json`.
Replace them with your own entries. The special registrar entry `none` leaves
delegation unmanaged.

## Typical Delegations

### Same provider for REG and DNS

Purpose: Use the same provider as a registrar and DNS service.

Why? Simplicity.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("namedotcom_main"),
  SERVICE("namedotcom_main"),
  A("@", "10.2.3.4"),
);
```
{% endcode %}

### Different provider for REG and DNS

Purpose: Use one provider as registrar, a different for DNS service.

Why? Some registrars do not provide DNS server, or their service is sub-standard and you want to use a high-performance DNS server.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("namedotcom_main"),
  SERVICE("aws_main"),
  A("@", "10.2.3.4"),
);
```
{% endcode %}

### Registrar is elsewhere

Purpose: This is a "DNS only" configuration.  Use it when you don't control the registrar but you do control the DNS records.

Why? You don't have access to the registrar, or the registrar is not supported by DNSControl. However you do have API access for updating the zone's records (most likely at a different provider).

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("none"),
  SERVICE("namedotcom_main"),
  A("@", "10.2.3.4"),
);
```
{% endcode %}

### Domain is "nowhere"

Suppose you don't want to manage a domain, but you want to list the zone in your `dnsconfig.js` file for inventory purposes. For example, suppose there are domains that some other part of your company maintains, but you want to list it in your `dnsconfig.js` because it is authoritative for the company.

```javascript
function INVENTORY_ONLY(name) {
    D(name, REGISTRAR("none"), { no_ns: "true" });
}

INVENTORY_ONLY("example.com");
INVENTORY_ONLY("example2.com");
INVENTORY_ONLY("example.net");
```

Now you can produce a list of your zones like this:

```shell
dnscontrol print-ir | jq -r '.domains[].name'
```

### Zone is elsewhere

Purpose: This is a "Registrar only" configuration.  Use it when you control the registrar but want to delegate the zone to someone else.

Why? We are delegating the domain to someone else. In this example we're pointing the domain to the nsone.net DNS service, which someone else is controlling.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("namedotcom_main"),
  NAMESERVER("dns1.p03.nsone.net."),
  NAMESERVER("dns2.p03.nsone.net."),
  NAMESERVER("dns3.p03.nsone.net."),
  NAMESERVER("dns4.p03.nsone.net."),
);
```
{% endcode %}

### Override nameservers

Purpose: Ignore the provider's default nameservers and substitute our own.

Why? Rarely used unless the DNS provider's API does not support querying what the nameservers are, or the API is returning invalid data, or if the API returns no information.  Sometimes APIs return no (useful) information when the domain is new; this is a good temporary work-around until the API starts working.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("namedotcom_main"),
  SERVICE("cloudflare_main", 0),  // Set the DNS provider but ignore the nameservers it suggests (0 == take none of the names it reports)
  NAMESERVER("kim.ns.cloudflare.com."),
  NAMESERVER("walt.ns.cloudflare.com."),
  A("@", "10.2.3.4"),
);
```
{% endcode %}

### Add nameservers

Purpose: Use the default nameservers from the registrar but add additional ones.

Why? Usually only to correct a bug or misconfiguration elsewhere.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("namedotcom_main"),
  SERVICE("namedotcom_main"),
  NAMESERVER("ns1.myexample.com"),
  A("@", "10.2.3.4"),
);
```
{% endcode %}

### Shadow nameservers

Purpose: Secretly publish your DNS zone records to another server.

Why? There are many reasons to do this:

- You are preparing to move to a different DNS provider and want to test it before you cut over.
- You want your DNS records stored somewhere else in case you have to switch over in an emergency.
- You are sending the zone to a local caching DNS server.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("namedotcom_main"),
  SERVICE("namedotcom_main"), // Our real DNS server
  SERVICE("cloudflare_main", 0), // Quietly send a copy of the zone here.
  SERVICE("bind", 0), // And here too!
  A("@", "10.2.3.4"),
);
```
{% endcode %}

### Dual DNS Providers

Purpose: Use two different DNS services:

Why? Diversity. If one DNS provider goes down, the other will be used.

Little known fact: Most DNS recursive resolvers monitor which DNS servers are performing the best and automatically start avoiding servers that are slow or down. This means that if you use this technique and one DNS provider goes down, after a while your users won't be affected.  Not all software does this properly. More info: https://www.dns-oarc.net/files/workshop-201203/OARC-workshop-London-2012-NS-selection.pdf

{% hint style="info" %}
**NOTE**: This is overkill unless you have millions of users and strict up-time requirements.
{% endhint %}

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("namedotcom_main"),
  SERVICE("aws_main", 2),  // Take 2 nameservers from AWS
  SERVICE("gcp_main", 2),  // Take 2 nameservers from GCP
  A("@", "10.2.3.4"),
);
```
{% endcode %}

## Other uses

### Make zonefile backups

Purpose: Make backups of DNS records in a zone.  This generates a zonefile listing all the records in the zone.

Why? You want to write out a BIND-style zonefile for debugging, historical, or auditing purposes. Some sites do backups of these zonefiles to create a history of changes. This is different than keeping a history of `dnsconfig.js` because this is the output of DNSControl, not the input.

{% hint style="danger" %}
**NOTE**: This won't work if you use pseudo rtypes that BIND doesn't support.
{% endhint %}

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("namedotcom_main"),
  SERVICE("namedotcom_main"),
  SERVICE("bind", 0), // Don't activate any nameservers related to BIND.
  A("@", "10.2.3.4"),
);
```
{% endcode %}

### Monitor delegation

Purpose: You don't control the registrar but want to detect if the delegation changes. You can specify the existing nameservers in `dnsconfig.js` and you will get a notification if the delegation diverges.

Why? Sometimes you just want to know if something changes!

See the [DNS-over-HTTPS Provider](../provider/dnsoverhttps.md) documentation for more info.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("DNS-over-HTTPS"),
  NAMESERVER("ns1.example.com."),
  NAMESERVER("ns2.example.com."),
);
```
{% endcode %}

{% hint style="info" %}
**NOTE**: This checks the NS records via a DNS query.  It does not check the
registrar's delegation (i.e. the `Name Server:` field in whois). In theory
these are the same thing but there may be situations where they are not.
{% endhint %}

## Helper macros

DNSControl has some built-in macros that you might find useful.

### `DOMAIN_ELSEWHERE`

Easily delegate a domain to a specific list of nameservers.

{% code title="dnsconfig.js" %}

```javascript
DOMAIN_ELSEWHERE("example.com", REGISTRAR("namedotcom_main"), [
    "dns1.example.net.",
    "dns2.example.net.",
    "dns3.example.net.",
]);
```

{% endcode %}

### `DOMAIN_ELSEWHERE_AUTO`

Easily delegate a domain to a nameserver via an API query.

This is similar to `DOMAIN_ELSEWHERE` but the list of nameservers is queried from the API of a single DNS provider.

{% code title="dnsconfig.js" %}
```javascript
DOMAIN_ELSEWHERE_AUTO("example.com", REGISTRAR("namedotcom_main"), SERVICE("aws_main"));
DOMAIN_ELSEWHERE_AUTO("example2.com", REGISTRAR("namedotcom_main"), SERVICE("gcp_main"));
```
{% endcode %}

## Limits

{% hint style="info" %}
**NOTE**: Not all providers allow full control over the NS records of your zone. It is not recommended to use these providers in complicated scenarios such as hosting across multiple providers. See individual provider docs for more info.
{% endhint %}
