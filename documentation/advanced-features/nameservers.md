# Nameservers and Delegations

DNSControl can handle a variety of provider scenarios. The registrar and DNS provider can be the same company, different company, they can even be unknown! The document shows examples of many common and uncommon configurations.

# Constants

All the examples use the variables.  Substitute your own.

{% code title="dnsconfig.js" %}
```javascript
// ========== Registrars:

// A typical registrar.
var REG_NAMECOM = NewRegistrar("namedotcom_main");

// The "NONE" registrar is a "fake" registrar.
// This is useful if the registrar is not supported by DNSControl,
// or if you don't want to control the domain's delegation.
var REG_NONE = NewRegistrar("none");

// ========== DNS Providers:

var DNS_NAMECOM = NewDnsProvider("namedotcom_main");
var DNS_AWS = NewDnsProvider("aws_main");
var DNS_GOOGLE = NewDnsProvider("gcp_main");
var DNS_CLOUDFLARE = NewDnsProvider("cloudflare_main");
var DNS_BIND = NewDnsProvider("bind");
```
{% endcode %}

# Typical Delegations

## Same provider for REG and DNS

Purpose: Use the same provider as a registrar and DNS service.

Why? Simplicity.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NAMECOM,
  DnsProvider(DNS_NAMECOM),
  A("@", "10.2.3.4"),
);
```
{% endcode %}

## Different provider for REG and DNS

Purpose: Use one provider as registrar, a different for DNS service.

Why? Some registrars do not provide DNS server, or their service is sub-standard and you want to use a high-performance DNS server.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NAMECOM,
  DnsProvider(DNS_AWS),
  A("@", "10.2.3.4"),
);
```
{% endcode %}

## Registrar is elsewhere

Purpose: This is a "DNS only" configuration.  Use it when you don't control the registrar but you do control the DNS records.

Why? You don't have access to the registrar, or the registrar is not supported by DNSControl. However you do have API access for updating the zone's records (most likely at a different provider).

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
D("example.com", REG_NONE,
  DnsProvider(DNS_NAMECOM),
  A("@", "10.2.3.4"),
);
```
{% endcode %}

## Domain is "nowhere"

Suppose you don't want to manage a domain, but you want to list the zone in your `dnsconfig.js` file for inventory purposes. For example, suppose there are domains that some other part of your company maintains, but you want to list it in your `dnsconfig.js` because it is authoritative for the company.

```javascript
var REG_NONE = NewRegistrar("none");

function INVENTORY_ONLY(name) {
    D(name, REG_NONE, { no_ns: "true" });
}

INVENTORY_ONLY('example.com");
INVENTORY_ONLY('example2.com");
INVENTORY_ONLY('example.net");
```

Now you can produce a list of your zones like this:

```shell
dnscontrol print-ir | jq -r '.domains[].name'
```

## Zone is elsewhere

Purpose: This is a "Registrar only" configuration.  Use it when you control the registrar but want to delegate the zone to someone else.

Why? We are delegating the domain to someone else. In this example we're pointing the domain to the nsone.net DNS service, which someone else is controlling.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NAMECOM,
  NAMESERVER("dns1.p03.nsone.net."),
  NAMESERVER("dns2.p03.nsone.net."),
  NAMESERVER("dns3.p03.nsone.net."),
  NAMESERVER("dns4.p03.nsone.net."),
);
```
{% endcode %}

## Override nameservers

Purpose: Ignore the provider's default nameservers and substitute our own.

Why? Rarely used unless the DNS provider's API does not support querying what the nameservers are, or the API is returning invalid data, or if the API returns no information.  Sometimes APIs return no (useful) information when the domain is new; this is a good temporary work-around until the API starts working.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NAMECOM,
  DnsProvider(DNS_CLOUDFLARE, 0),  // Set the DNS provider but ignore the nameservers it suggests (0 == take none of the names it reports)
  NAMESERVER("kim.ns.cloudflare.com."),
  NAMESERVER("walt.ns.cloudflare.com."),
  A("@", "10.2.3.4"),
);
```
{% endcode %}

## Add nameservers

Purpose: Use the default nameservers from the registrar but add additional ones.

Why? Usually only to correct a bug or misconfiguration elsewhere.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NAMECOM,
  DnsProvider(DNS_NAMECOM),
  NAMESERVER("ns1.myexample.com"),
  A("@", "10.2.3.4"),
);
```
{% endcode %}

## Shadow nameservers

Purpose: Secretly publish your DNS zone records to another server.

Why? There are many reasons to do this:

* You are preparing to move to a different DNS provider and want to test it before you cut over.
* You want your DNS records stored somewhere else in case you have to switch over in an emergency.
* You are sending the zone to a local caching DNS server.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NAMECOM,
  DnsProvider(DNS_NAMECOM), // Our real DNS server
  DnsProvider(DNS_CLOUDFLARE, 0), // Quietly send a copy of the zone here.
  DnsProvider(DNS_BIND, 0), // And here too!
  A("@", "10.2.3.4"),
);
```
{% endcode %}

## Dual DNS Providers

Purpose: Use two different DNS services:

Why? Diversity. If one DNS provider goes down, the other will be used.

Little known fact: Most DNS recursive resolvers monitor which DNS servers are performing the best and automatically start avoiding servers that are slow or down. This means that if you use this technique and one DNS provider goes down, after a while your users won't be affected.  Not all software does this properly. More info: https://www.dns-oarc.net/files/workshop-201203/OARC-workshop-London-2012-NS-selection.pdf

{% hint style="info" %}
**NOTE**: This is overkill unless you have millions of users and strict up-time requirements.
{% endhint %}

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NAMECOM,
  DnsProvider(DNS_AWS, 2),  // Take 2 nameservers from AWS
  DnsProvider(DNS_GOOGLE, 2),  // Take 2 nameservers from GCP
  A("@", "10.2.3.4"),
);
```
{% endcode %}

# Other uses

## Make zonefile backups

Purpose: Make backups of DNS records in a zone.  This generates a zonefile listing all the records in the zone.

Why? You want to write out a BIND-style zonefile for debugging, historical, or auditing purposes. Some sites do backups of these zonefiles to create a history of changes. This is different than keeping a history of `dnsconfig.js` because this is the output of DNSControl, not the input.

{% hint style="danger" %}
**NOTE**: This won't work if you use pseudo rtypes that BIND doesn't support.
{% endhint %}

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NAMECOM,
  DnsProvider(DNS_NAMECOM),
  DnsProvider(DNS_BIND, 0), // Don't activate any nameservers related to BIND.
  A("@", "10.2.3.4"),
);
```
{% endcode %}

## Monitor delegation

Purpose: You don't control the registrar but want to detect if the delegation changes. You can specify the existing nameservers in `dnsconfig.js` and you will get a notification if the delegation diverges.

Why? Sometimes you just want to know if something changes!

See the [DNS-over-HTTPS Provider](../provider/dnsoverhttps.md) documentation for more info.

{% code title="dnsconfig.js" %}
```javascript
var REG_MONITOR = NewRegistrar("DNS-over-HTTPS");

D("example.com", REG_MONITOR,
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

# Helper macros

DNSControl has some built-in macros that you might find useful.

## `DOMAIN_ELSEWHERE`

Easily delegate a domain to a specific list of nameservers.

{% code title="dnsconfig.js" %}
```javascript
DOMAIN_ELSEWHERE("example.com", REG_NAMECOM, [
    "dns1.example.net.",
    "dns2.example.net.",
    "dns3.example.net.",
]);
```
{% endcode %}

## `DOMAIN_ELSEWHERE_AUTO`

Easily delegate a domain to a nameserver via an API query.

This is similar to `DOMAIN_ELSEWHERE` but the list of nameservers is queried from the API of a single DNS provider.

{% code title="dnsconfig.js" %}
```javascript
DOMAIN_ELSEWHERE_AUTO("example.com", REG_NAMECOM, DNS_AWS);
DOMAIN_ELSEWHERE_AUTO("example2.com", REG_NAMECOM, DNS_GOOGLE);
```
{% endcode %}

# Limits

{% hint style="info" %}
**NOTE**: Not all providers allow full control over the NS records of your zone. It is not recommended to use these providers in complicated scenarios such as hosting across multiple providers. See individual provider docs for more info.
{% endhint %}
