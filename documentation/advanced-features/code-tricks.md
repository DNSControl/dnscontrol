# "Builders"

Problem: It is difficult to get CAA and other records exactly right.

Solution: Use a "builder" to construct it for you.

* [CAA_BUILDER](../language-reference/domain-modifiers/CAA_BUILDER.md)
* [DKIM_BUILDER](../language-reference/domain-modifiers/DKIM_BUILDER.md)
* [DMARC_BUILDER](../language-reference/domain-modifiers/DMARC_BUILDER.md)
* [M365_BUILDER](../language-reference/domain-modifiers/M365_BUILDER.md)
* [SPF_BUILDER](../language-reference/domain-modifiers/SPF_BUILDER.md)

# Trailing commas

You might encounter `D()` statements in code examples that include `END` at the end, such as:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  A("test", "1.2.3.4"),
END);
```
{% endcode %}

As of [DNSControl v4.15.0](https://github.com/DNSControl/dnscontrol/releases/tag/v4.15.0), the `END` statements are no longer necessary. These were originally included for historical reasons that are now irrelevant. You can safely remove them from your configurations.

# Repeat records in many domains (macros)

Problem: I have a set of records I'd like to include in many domains.

Solution: Assign a list to a var and use the var in each domain.

Example:

Domains that use Google G-Suite require a specific list of MX records, plus there are some CNAMEs that are useful (but we only want the CNAMEs on a subset of domains).

{% code title="dnsconfig.js" %}
```javascript
var GOOGLE_APPS_DOMAIN_MX = [
  MX("@", 1, "aspmx.l.google.com."),
  MX("@", 5, "alt1.aspmx.l.google.com."),
  MX("@", 5, "alt2.aspmx.l.google.com."),
  MX("@", 10, "alt3.aspmx.l.google.com."),
  MX("@", 10, "alt4.aspmx.l.google.com."),
];
var GOOGLE_APPS_DOMAIN_SITES = [
  CNAME("groups", "ghs.googlehosted.com."),
  CNAME("drive", "ghs.googlehosted.com."),
  CNAME("calendar", "ghs.googlehosted.com."),
  CNAME("mail", "ghs.googlehosted.com."),
  CNAME("sites", "ghs.googlehosted.com."),
  CNAME("start", "ghs.googlehosted.com."),
];

D("primarydomain.tld", REG_NAMECOM, DnsProvider(DSP_MY_PROVIDER),
   GOOGLE_APPS_DOMAIN_MX,
   GOOGLE_APPS_DOMAIN_SITES,
   A(...),
   CNAME(...)
}

D("aliasdomain.tld", REG_NAMECOM, DnsProvider(DSP_MY_PROVIDER),
   GOOGLE_APPS_DOMAIN_MX,
   // FYI: GOOGLE_APPS_DOMAIN_SITES is not used here.
   A(...),
   CNAME(...)
}
```
{% endcode %}

# Many domains with the exact same records

Problem: We have many domains, each should have the exact same records.

Solution 1: Use a macro.

```javascript
function PARKED_R53(name) {
    D(name, REG_NAMECOM, DnsProvider(DSP_MY_PROVIDER),
       A("@", "10.2.3.4"),
       CNAME("www", "@"),
        SPF_NONE, //deters spammers from using the domain in From: lines.
        );
}

PARKED_R53("example1.tld");
PARKED_R53("example2.tld");
PARKED_R53("example3.tld");
```

Solution 2: Use a loop. (Note: See caveats below.)

{% code title="dnsconfig.js" %}
```javascript
// The domains are parked. Use the exact same records for each.
_.each(
  [
    "example1.tld",
    "example2.tld",
    "example3.tld",
  ],
  function (d) {
    D(d, REG_NAMECOM, DnsProvider(DSP_MY_PROVIDER),
       A("@", "10.2.3.4"),
       CNAME("www", "@"),
    );
  }
);
```
{% endcode %}

# Caveats about getting too fancy

The `dnsconfig.js` language is JavaScript. On the plus side, this means you can use loops and variables and anything else you want.

However, we don't recommend you get too fancy.

*A new JS interpreter may break your code*

Some day we may change from the [Otto JS interpreter](https://github.com/robertkrimen/otto) to something else. This may break your configuration if you depend on unusual or obscure behavior of Otto.

Loops and macros are fine. Just don't get too fancy.

*Complexity is a killer*

As Brian Kernighan wrote, "Debugging is twice as hard as writing the code in the first place. Therefore, if you write the code as cleverly as possible, you are, by definition, not smart enough to debug it."

Sure, you can do a lot of neat tricks with `if/then`s and macros and loops. Yes, YOU understand the code. However, think about your coworkers who will be the *next* person to edit the file. Are you setting them up for failure?

And what about You Six Months From Now (YSMFN)? Have you met YSMFN? They're a great person. You'll meet them soon. In about 6 months, I predict. That person is a lot like you, but definitely won't remember how all those clever tricks work. That person also might be tired, sleepy, and possibly drunk. You really want to keep the configuration file simple for YSMFN.

The goal of DNSControl is to empower non-experts to safely make DNS changes. A DNS expert should create the initial configuration, but your non-expert coworkers should be able to send PR and make changes without too much hand-holding. Complexity prevents that.

*What to do instead?*

Isolate the clever stuff from what a typical user will need to edit.

For example, Stack Overflow, puts all macro definitions and fancy stuff at the top of the file. The domains are later in the file.

We name the macros to be easy to understand for the user. For example, we have a few macros named `SPF_NONE`, `SPF_GSUITE`, and `SPF_domain_tld` (where `domain_tld` is the name of a domain). I bet you can guess which to use for a new domain without seeing the definition.

We also comment extensively. Most records have a comment at the end of the line with the ticket number of the request related to the record. Before each domain there is a long comment explaining why the domain exists, who requested it, any associated ticket numbers, and so on.

We also comment the individual parts of a record. Look at the [SPF Optimizer](../language-reference/domain-modifiers/SPF_BUILDER.md) example. Each part of the SPF record has a comment.
