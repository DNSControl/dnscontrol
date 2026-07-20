---
name: D_EXTEND
parameters:
  - name
  - modifiers...
parameter_types:
  name: string
  "modifiers...": DomainModifier[]
---

`D_EXTEND` adds records (and metadata) to a domain previously defined by [`D()`](D.md). It can also be used to add subdomain records (and metadata) to a previously defined domain.

The first argument is a domain name. If it exactly matches a previously defined domain, `D_EXTEND()` behaves the same as [`D()`](D.md), simply adding records as if they had been specified in the original [`D()`](D.md).

If the domain name does not match an existing domain, but could be a (non-delegated) subdomain of an existing domain, the new records (and metadata) are added with the subdomain part appended to all record names (labels), and targets (as appropriate). See the examples below.

Matching the domain name to previously-defined domains is done using a `longest match` algorithm. If `domain.tld` and `sub.domain.tld` are defined as separate domains via separate [`D()`](D.md) statements, then `D_EXTEND("sub.sub.domain.tld", ...)` would match `sub.domain.tld`, not `domain.tld`.

Some operators only act on an apex domain (e.g. [`CF_SINGLE_REDIRECT`](../domain-modifiers/CF_SINGLE_REDIRECT.md), [`CF_REDIRECT`](../domain-modifiers/CF_REDIRECT.md), and [`CF_TEMP_REDIRECT`](../domain-modifiers/CF_TEMP_REDIRECT.md)). Using them in a `D_EXTEND` subdomain may not be what you expect.

{% code title="dnsconfig.js" %}
```javascript
D("domain.tld", REG_MY_PROVIDER, DnsProvider(DNS),
  A("@", "127.0.0.1"), // domain.tld
  A("www", "127.0.0.2"), // www.domain.tld
  CNAME("a", "b"), // a.domain.tld -> b.domain.tld
);
D_EXTEND("domain.tld",
  A("aaa", "127.0.0.3"), // aaa.domain.tld
  CNAME("c", "d"), // c.domain.tld -> d.domain.tld
);
D_EXTEND("sub.domain.tld",
  A("bbb", "127.0.0.4"), // bbb.sub.domain.tld
  A("ccc", "127.0.0.5"), // ccc.sub.domain.tld
  CNAME("e", "f"), // e.sub.domain.tld -> f.sub.domain.tld
);
D_EXTEND("sub.sub.domain.tld",
  A("ddd", "127.0.0.6"), // ddd.sub.sub.domain.tld
  CNAME("g", "h"), // g.sub.sub.domain.tld -> h.sub.sub.domain.tld
);
D_EXTEND("sub.domain.tld",
  A("@", "127.0.0.7"), // sub.domain.tld
  CNAME("i", "j"), // i.sub.domain.tld -> j.sub.domain.tld
);
```
{% endcode %}

This will end up in the following modifications: (This output assumes the `--full` flag)

```text
******************** Domain: domain.tld
----- Getting nameservers from: cloudflare
----- DNS Provider: cloudflare...7 corrections
#1: CREATE A aaa.domain.tld 127.0.0.3
#2: CREATE A bbb.sub.domain.tld 127.0.0.4
#3: CREATE A ccc.sub.domain.tld 127.0.0.5
#4: CREATE A ddd.sub.sub.domain.tld 127.0.0.6
#5: CREATE A sub.domain.tld 127.0.0.7
#6: CREATE A www.domain.tld 127.0.0.2
#7: CREATE A domain.tld 127.0.0.1
#8: CREATE CNAME a.domain.tld b.domain.tld.
#9: CREATE CNAME c.domain.tld d.domain.tld.
#10: CREATE CNAME e.sub.domain.tld f.sub.domain.tld.
#11: CREATE CNAME g.sub.sub.domain.tld h.sub.sub.domain.tld.
#12: CREATE CNAME i.sub.domain.tld j.sub.domain.tld.
```

ProTips: `D_EXTEND()` permits you to create very complex and sophisticated configurations, but you shouldn't. Be nice to the next person that edits the file, who may not be as expert as yourself. Enhance readability by putting any `D_EXTEND()` statements immediately after the original [`D()`](D.md), like in above example. Avoid the temptation to obscure the addition of records to existing domains with randomly placed `D_EXTEND()` statements. Don't build up a domain using loops of `D_EXTEND()` statements. You'll be glad you didn't.
