---
name: SPF_BUILDER
parameters:
  - label
  - overflow
  - overhead1
  - raw
  - ttl
  - txtMaxSize
  - parts
  - flatten
parameters_object: true
parameter_types:
  label: string?
  overflow: string?
  overhead1: string?
  raw: string?
  ttl: Duration?
  txtMaxSize: number?
  parts: string[]
  flatten: string[]?
---

DNSControl can optimize the SPF settings on a domain by flattening (inlining) includes and removing duplicates. DNSControl also makes it easier to document your SPF configuration.

{% hint style="warning" %}
**WARNING**: Flattening SPF includes is risky.  Only flatten an SPF
setting if it is absolutely needed to bring the number of "lookups"
to be less than 10. In fact, it is debatable whether or not ISPs
enforce the "10 lookup rule".
{% endhint %}

## The old way

Here is an example of how SPF settings are normally done:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  TXT("v=spf1 ip4:198.252.206.0/24 ip4:192.111.0.0/24 include:_spf.google.com include:mailgun.org include:spf-basic.fogcreek.com include:mail.zendesk.com include:servers.mcsv.net include:sendgrid.net include:450622.spf05.hubspotemail.net ~all"),
);
```
{% endcode %}

This has a few problems:

* No comments. It is difficult to add a comment. In particular, we want to be able to list which ticket requested each item in the SPF setting so that history is retained.
* Ugly diffs.  If you add an element to the SPF setting, the diff will show the entire line changed, which is difficult to read.
* Too many lookups. The SPF RFC says that SPF settings should not require more than 10 DNS lookups. If we manually flatten (i.e. "inline") an include, we have to remember to check back to see if the settings have changed. Humans are not good at that kind of thing.

## The DNSControl way

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
      "include:mailgun.org", // Greenhouse.io
      "include:spf-basic.fogcreek.com", // Fogbugz
      "include:mail.zendesk.com", // Zenddesk
      "include:servers.mcsv.net", // MailChimp
      "include:sendgrid.net", // SendGrid
      "include:450622.spf05.hubspotemail.net", // Hubspot (Ticket# SREREQ-107)
      "~all"
    ],
    flatten: [
      "spf-basic.fogcreek.com", // Rationale: Being deprecated. Low risk if it breaks.
      "450622.spf05.hubspotemail.net" // Rationale: Unlikely to change without warning.
    ]
  }),
);
```
{% endcode %}

By using the `SPF_BUILDER()` we gain many benefits:

* Comments can appear next to the element they refer to.
* Diffs will be shorter and more specific; therefore easier to read.
* Automatic flattening.  We can specify which includes should be flattened and DNSControl will do the work. It will even warn us if the includes change.

## Syntax

When you want to specify SPF settings for a domain, use the `SPF_BUILDER()` function.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  ...
  ...
  ...
  SPF_BUILDER({
    label: "@",
    overflow: "_spf%d",  // Delete this line if you don't want big strings split.
    overhead1: "20",  // There are 20 bytes of other TXT records on this domain.  Compensate for this.
    raw: "_rawspf",  // Delete this line if the default is sufficient.
    parts: [
      "v=spf1",
      // fill in your SPF items here
      "~all"
    ],
    flatten: [
      // fill in any domains to inline.
    ]
  }),
  ...
  ...
);
```
{% endcode %}

The parameters are:

* `label:` The label of the first TXT record. (Optional. Default: `"@"`)
* `overflow:` If set, SPF strings longer than 255 chars will be split into multiple TXT records. The value of this setting determines the template for what the additional labels will be named. If not set, no splitting will occur and DNSControl may generate TXT strings that are too long.
* `overhead1:` "Overhead for the 1st TXT record".  When calculating the max length of each TXT record, reduce the maximum for the first TXT record in the chain by this amount.
* `raw:` The label of the unaltered SPF settings. Setting to an empty string `''` will disable this. (Optional. Default: `"_rawspf"`)
* `ttl:` This allows setting a specific TTL on this SPF record. (Optional. Default: using default record TTL)
* `txtMaxSize` The maximum size for each TXT record. Values over 255 will result in [multiple strings][multi-string]. General recommendation is to [not go higher than 450][record-size] so that DNS responses will still fit in a UDP packet. (Optional. Default: `"255"`)
* `parts:` The individual parts of the SPF settings.
* `flatten:` Which includes should be inlined. For safety purposes the flattening is done on an opt-in basis. If `"*"` is listed, all includes will be flattened... this might create more problems than is solves due to length limitations.

[multi-string]: https://tools.ietf.org/html/rfc4408#section-3.1.3 [record-size]: https://tools.ietf.org/html/rfc4408#section-3.1.4

`SPF_BUILDER()` returns multiple `TXT()` records:

  * `TXT("@", "v=spf1 .... ~all")`
    *  This is the optimized configuration.
  * `TXT("_spf1", "...")`
    * If the optimizer needs to split a long string across multiple TXT records, the additional TXT records will have labels `_spf1`, `_spf2`, `_spf3`, etc.
  * `TXT("_rawspf", "v=spf1 .... ~all")`
    * This is the unaltered SPF configuration. This is purely for debugging purposes and is not used by any email or anti-spam system.  It is only generated if flattening is requested.

We recommend first using this without any flattening. Make sure `dnscontrol preview` works as expected. Once that is done, add the flattening required to reduce the number of lookups to 10 or less.

To count the number of lookups, you can use our interactive SPF debugger at [https://dnscontrol.github.io/dnscontrol/flattener/index.html](https://dnscontrol.github.io/dnscontrol/flattener/index.html)

# The first in a chain is special

When generating the chain of SPF records, each one is max length 255.  For the first item in the chain, the max is 255 - "overhead1".  Setting this to 255 or higher has undefined behavior.

Why is this useful?

Some sites desire having all DNS queries fit in a single packet so that UDP, not TCP, can be used to satisfy all requests. That means all responses have to be relatively small.

When an SPF system does a "TXT" lookup, it gets SPF and non-SPF records.  This makes the first link in the chain extra large.

The bottom line is that if you want the TXT records to fit in a UDP packet, keep increasing the value of `overhead1` until the packet is no longer truncated.

Example:

```shell
dig +short whatexit.org txt | wc -c
   118
```

Setting `overhead1` to 118 should be sufficient.

```shell
dig +short stackoverflow.com txt | wc -c
     582
```

Since 582 is bigger than 255, it might not be possible to achieve the goal.  Any value larger than 255 will disable all flattening.  Try 170, then 180, 190 until you get the desired results.

A validator such as [https://www.kitterman.com/spf/validate.html](https://www.kitterman.com/spf/validate.html) will tell you if the queries are being truncated and TCP was required to get the entire record. (Sadly it caches heavily.)

## Notes about the `spfcache.json`

DNSControl will optionally keep a cache of the DNS lookups performed during optimization.  In the event that a DNS server is down, the cache will be used. This makes it possible to do `dnscontrol push` even if your or third-party DNS servers are down.

To enable this feature, create an (empty) file called `spfcache.json` in the current directory.  To disable this feature, delete the file. There are no command-line flags related to this feature.

The `spfcache.json` stored the cached DNS lookups. If it needs to be updated, the new file contents will be written to a file called `spfcache.updated.json` and instructions such as the ones below will be output telling you exactly what to do:

```shell
dnscontrol preview
1 Validation errors:
WARNING: 2 spf record lookups are out of date with cache (_spf.google.com,_netblocks3.google.com).
Wrote changes to spfcache.updated.json. Please rename and commit:
    $ mv spfcache.updated.json spfcache.json
    $ git commit spfcache.json
```

In this case, you are being asked to replace `spfcache.json` with the newly generated data in `spfcache.updated.json`.

The instructions are hardcoded strings. The filenames will not change. The instructions assume you use git. If you use something else, please do the appropriate equivalent command.

## Caveats

1. DNSControl 'gives up' if it sees SPF records it can't understand. This includes: syntax errors, features that our spflib doesn't know about, overly complex SPF settings, and anything else that we we didn't feel like implementing.

2. The TXT record that is generated may exceed DNS limits.  dnscontrol will not generate a single TXT record that exceeds DNS limits, but it ignores the fact that there may be other TXT records on the same label.  For example, suppose it generates a TXT record on the bare domain (stackoverflow.com) that is 250 bytes long. That's fine and doesn't require a continuation record.  However if there is another TXT record (not an SPF record, perhaps a TXT record used to verify domain ownership), the total packet size of all the TXT records could exceed 512 bytes, and will require EDNS or a TCP request.

3. DNSControl does not warn if the number of lookups exceeds 10. We hope to implement this some day.

4. The `redirect=` directive is only partially implemented.  We only handle the case where redirect is the last item in the SPF record. In which case, it is equivalent to `include:`.

## Advanced Technique: Interactive SPF Debugger

DNSControl includes an experimental system for viewing SPF settings:

[https://dnscontrol.github.io/dnscontrol/flattener/index.html](https://dnscontrol.github.io/dnscontrol/flattener/index.html)

You can also run this locally (it is self-contained) by opening `dnscontrol/docs/flattener/index.html` in your browser.

You can use this to determine the minimal number of domains you need to flatten to have fewer than 10 lookups.

The output is as follows:

1. The top part lists the domain as it current is configured, how many lookups it requires, and includes a checkbox for each item that could be flattened.

2. Fully flattened: This section shows the SPF configuration if you fully flatten it. i.e. This is what it would look like if all the checkboxes were checked. Note that this result is likely to be longer than 255 bytes, the limit for a single TXT string.

3. Fully flattened split: This takes the "fully flattened" result and splits it into multiple DNS records.  To continue to the next record an include is added.

## Advanced Technique: Define once, use many

In some situations we define an SPF setting once and want to reuse it on many domains. Here's how to do this:

{% code title="dnsconfig.js" %}
```javascript
var SPF_MYSETTINGS = SPF_BUILDER({
  label: "@",
  overflow: "_spf%d",
  raw: "_rawspf",
  parts: [
    "v=spf1",
    ...
    "~all"
  ],
  flatten: [
    ...
  ]
});

D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    SPF_MYSETTINGS,
);

D("example2.tld", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
     SPF_MYSETTINGS,
);
```
{% endcode %}
