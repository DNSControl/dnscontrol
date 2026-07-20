---
name: D
parameters:
  - name
  - registrar
  - modifiers...
parameter_types:
  name: string
  registrar: string
  "modifiers...": DomainModifier[]
---

`D` adds a new Domain for DNSControl to manage. The first two arguments are required: the domain name (fully qualified `example.com` without a trailing dot), and the name of the registrar (as previously declared with [NewRegistrar](NewRegistrar.md)). Any number of additional arguments may be included to add DNS Providers with [DNSProvider](NewDnsProvider.md), add records with [A](../domain-modifiers/A.md), [CNAME](../domain-modifiers/CNAME.md), and so forth, or add metadata.

Modifier arguments are processed according to type as follows:

- A function argument will be called with the domain object as it's only argument. Most of the [built-in modifier functions](https://docs.dnscontrol.org/language-reference/domain-modifiers) return such functions.
- An object argument will be merged into the domain's metadata collection.
- An array argument will have all of it's members evaluated recursively. This allows you to combine multiple common records or modifiers into a variable that can be used like a macro in multiple domains.

{% code title="dnsconfig.js" %}
```javascript
// simple domain
D("example.com", REG_MY_PROVIDER,
  DnsProvider(DSP_MY_PROVIDER),
  A("@","1.2.3.4"),           // "@" means the apex domain. In this case, "example.com" itself.
  CNAME("test", "foo.example2.com."),
);

// "macro" for records that can be mixed into any zone
var GOOGLE_APPS_DOMAIN_MX = [
    MX("@", 1, "aspmx.l.google.com."),
    MX("@", 5, "alt1.aspmx.l.google.com."),
    MX("@", 5, "alt2.aspmx.l.google.com."),
    MX("@", 10, "alt3.aspmx.l.google.com."),
    MX("@", 10, "alt4.aspmx.l.google.com."),
]

D("other-example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  A("@","1.2.3.4"),
  CNAME("test", "foo.example2.com."),
  GOOGLE_APPS_DOMAIN_MX,
);
```
{% endcode %}

{% hint style="info" %}
**What is "@"?** The label `@` is a special name that means the domain itself,
otherwise known as the domain's apex, the bare domain, or the naked domain.  
In other words, if you want to put a DNS record at the apex of a domain, use an `"@"` for the label, not an empty string (`""`).
In the above example, `example.com` has an `A` record with the value `"1.2.3.4"` at the apex of the domain.
{% endhint %}

# `no_ns`

To prevent DNSControl from accidentally deleting your nameservers (at the parent domain), registrar updates are disabled if the list of nameservers for a zone (as computed from `dnsconfig.js`) is empty.

This can happen when a provider doesn't give any control over the apex NS records, there are no default nameservers, there are no `NAMESERVER()` statements, and the provider returns an empty list of nameservers (such as Gandi and Vercel).

In this situation, you will see an error message such as:

```
Skipping registrar REGISTRAR: No nameservers declared for domain "example.com". Add {no_ns: "true"} to force
```

To add this, add the meta data to the zone immediately following the registrar.

```javascript
D("example.com", REG_MY_PROVIDER, {no_ns: "true"},
  ...
  ...
  ...
);
```

{% hint style="info" %}
**NOTE**: The value `true` of `no_ns` is a string.
{% endhint %}

# Split Horizon DNS

DNSControl supports Split Horizon DNS. Simply define the domain two or more times, each with their own unique parameters.

To differentiate the different domains, specify the domains as `domain.tld!tag`, such as `example.com!inside` and `example.com!outside`.

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DNS_INSIDE = NewDnsProvider("Cloudflare");
var DNS_OUTSIDE = NewDnsProvider("bind");

D("example.com!inside", REG_NONE, DnsProvider(DNS_INSIDE),
  A("www", "10.10.10.10"),
);

D("example.com!outside", REG_NONE, DnsProvider(DNS_OUTSIDE),
  A("www", "20.20.20.20"),
);

D_EXTEND("example.com!inside",
  A("internal", "10.99.99.99"),
);
```
{% endcode %}

A domain name without a `!` is assigned a tag that is the empty string. For example, `example.com` and `example.com!` are equivalent. However, we strongly recommend against using the empty tag, as it risks creating confusion.  In other words, if you have `domain.tld` and `domain.tld!external` you now require humans to remember that `domain.tld` is the external one.  I mean... the internal one.  You may have noticed this mistake, but will your coworkers?  Will you in six months? You get the idea.

DNSControl command line flag `--domains` matches the full name (with the "!").  If you define domains `example.com!john`, `example.com!paul`, and `example.com!george` then:

* `--domains=example.com` will not match any of the three.
* `--domains='example.com!george'` will only match george.
* `--domains='example.com!george,example.com!john'` will match george and john.
* `--domains='example.com!*'` will match all three.

{% hint style="info" %}
**NOTE**: The quotes are required if your shell treats `!` as a special
character, which is probably does.  If you see an error that mentions
`event not found` you probably forgot the quotes.
{% endhint %}
