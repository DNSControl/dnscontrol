# Examples

- [Examples](#examples)
  - [Typical DNS Records](#typical-dns-records)
  - [Set TTLs](#set-ttls)
  - [Variables for common IP Addresses](#variables-for-common-ip-addresses)
  - [Variables to swap active Data Center](#variables-to-swap-active-data-center)
  - [Macro for repeated records](#macro-for-repeated-records)
  - [Use SPF\_BUILDER to add comments to SPF records](#use-spf_builder-to-add-comments-to-spf-records)
  - [Set default records modifiers](#set-default-records-modifiers)
  - [Advanced Examples](#advanced-examples)
    - [Dual DNS Providers](#dual-dns-providers)
    - [Automate Fastmail DKIM records](#automate-fastmail-dkim-records)
    - [More advanced examples](#more-advanced-examples)

## Typical DNS Records

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    A("@", "1.2.3.4"),  // The naked or "apex" domain.
    A("server1", "2.3.4.5"),
    AAAA("wide", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"),
    CNAME("www", "server1"),
    CNAME("another", "service.mycloud.com."),
    MX("mail", 10, "mailserver"),
    MX("mail", 20, "mailqueue"),
    TXT("the", "message"),
    NS("department2", "ns1.dnsexample.com."), // use different nameservers
    NS("department2", "ns2.dnsexample.com."), // for department2.example.com
);
```
{% endcode %}

## Set TTLs

{% code title="dnsconfig.js" %}
```javascript
var mailTTL = TTL("1h");

D("example.com", REG_MY_PROVIDER,
    NAMESERVER_TTL("10m"), // On domain apex NS RRs
    DefaultTTL("5m"), // Default for a domain

    MX("@", 5, "1.2.3.4", mailTTL), // use variable to
    MX("@", 10, "4.3.2.1", mailTTL), // set TTL

    A("@", "1.2.3.4", TTL("10m")), // individual record
    CNAME("mail", "mx01"), // TTL of 5m, as defined per DefaultTTL()
);
```
{% endcode %}

## Variables for common IP Addresses

{% code title="dnsconfig.js" %}
```javascript
var addrA = IP("1.2.3.4")

var DSP_R53 = NewDnsProvider("route53_user1");

D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_R53),
    A("@", addrA), // 1.2.3.4
    A("www", addrA + 1), // 1.2.3.5
);
```
{% endcode %}

{% hint style="info" %}
**NOTE**: The [`IP()`](../language-reference/top-level-functions/IP.md) function doesn't currently support IPv6 (PRs welcome!).  IPv6 addresses are strings.
{% endhint %}
{% code title="dnsconfig.js" %}
```javascript
var addrAAAA = "0:0:0:0:0:0:0:0";
```
{% endcode %}

## Variables to swap active Data Center

{% code title="dnsconfig.js" %}
```javascript
var DSP_R53 = NewDnsProvider("route53_user1");

var dcA = IP("5.5.5.5");
var dcB = IP("6.6.6.6");

// switch to dcB to failover
var activeDC = dcA;

D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_R53),
    A("@", activeDC + 5), // fixed address based on activeDC
);
```
{% endcode %}

## Macro for repeated records

{% code title="dnsconfig.js" %}
```javascript
var GOOGLE_APPS_MX_RECORDS = [
    MX("@", 1, "aspmx.l.google.com."),
    MX("@", 5, "alt1.aspmx.l.google.com."),
    MX("@", 5, "alt2.aspmx.l.google.com."),
    MX("@", 10, "alt3.aspmx.l.google.com."),
    MX("@", 10, "alt4.aspmx.l.google.com."),
]

var GOOGLE_APPS_CNAME_RECORDS = [
    CNAME("calendar", "ghs.googlehosted.com."),
    CNAME("drive", "ghs.googlehosted.com."),
    CNAME("mail", "ghs.googlehosted.com."),
    CNAME("groups", "ghs.googlehosted.com."),
    CNAME("sites", "ghs.googlehosted.com."),
    CNAME("start", "ghs.googlehosted.com."),
]

D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_R53),
   GOOGLE_APPS_MX_RECORDS,
   GOOGLE_APPS_CNAME_RECORDS,
   A("@", "1.2.3.4"),
);
```
{% endcode %}

## Use SPF_BUILDER to add comments to SPF records

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  A("@", "10.2.2.2"),
  MX("@", "example.com."),
  SPF_BUILDER({
    label: "@",
    overflow: "_spf%d",
    raw: "_rawspf",
    ttl: "5m",
    parts: [
      "v=spf1",
      "ip4:198.252.206.0/24", // ny-mail*
      "ip4:192.111.0.0/24", // co-mail*
      "include:_spf.google.com", // GSuite
      "~all"
    ]
  }),
);
```
{% endcode %}

## Set default records modifiers

{% code title="dnsconfig.js" %}
```javascript
DEFAULTS(
    NAMESERVER_TTL("24h"),
    DefaultTTL("12h"),
    CF_PROXY_DEFAULT_OFF,
);
```
{% endcode %}

## Advanced Examples

### Dual DNS Providers

{% code title="dnsconfig.js" %}
```javascript

var DSP_R53 = NewDnsProvider("route53_user1");
var DSP_GCLOUD = NewDnsProvider("gcloud_admin");

D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_R53), DnsProvider(DSP_GCLOUD),
   A("@", "1.2.3.4"),
);

// above zone uses 8 NS records total (4 from each provider dynamically gathered)
// below zone will only take 2 from each for a total of 4. May be better for performance reasons.

D("example2.com", REG_MY_PROVIDER, DnsProvider(DSP_R53, 2), DnsProvider(DSP_GCLOUD ,2),
   A("@", "1.2.3.4"),
);

// or set a Provider as a non-authoritative backup (don"t register its nameservers)
D("example3.com", REG_MY_PROVIDER, DnsProvider(DSP_R53), DnsProvider(DSP_GCLOUD, 0),
   A("@", "1.2.3.4"),
);
```
{% endcode %}

### Automate Fastmail DKIM records

In this example we need a macro that can dynamically change for each domain.

Suppose you have many domains that use Fastmail as an MX. Here's a macro that sets the MX records.
{% code title="dnsconfig.js" %}
```javascript
var FASTMAIL_MX = [
  MX("@", 10, "in1-smtp.messagingengine.com."),
  MX("@", 20, "in2-smtp.messagingengine.com."),
]
```
{% endcode %}

Fastmail also supplied CNAMES to implement DKIM, and they all match a pattern
that includes the domain name. We can't use a simple macro. Instead, we use
a function that takes the domain name as a parameter to generate the right
records dynamically.
{% code title="dnsconfig.js" %}
```javascript
var FASTMAIL_DKIM = function(the_domain){
  return [
    CNAME("fm1._domainkey", "fm1." + the_domain + ".dkim.fmhosted.com."),
    CNAME("fm2._domainkey", "fm2." + the_domain + ".dkim.fmhosted.com."),
    CNAME("fm3._domainkey", "fm3." + the_domain + ".dkim.fmhosted.com."),
  ]
}
```
{% endcode %}

We can then use the macros as such:
{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_R53_MAIN = NewDnsProvider("r53_main");

D("example.com", REG_NONE, DnsProvider(DSP_R53_MAIN),
    FASTMAIL_MX,
    FASTMAIL_DKIM("example.com"),
);
```
{% endcode %}

### More advanced examples

See the [Code Tricks](../advanced-features/code-tricks.md) page.
