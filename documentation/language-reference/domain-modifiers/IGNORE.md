---
name: IGNORE
parameters:
    - labelSpec
    - typeSpec
    - targetSpec
parameter_types:
    labelSpec: string
    typeSpec: string?
    targetSpec: string?
---

`IGNORE()` makes it possible for DNSControl to share management of a domain with an external system.  The parameters of `IGNORE()` indicate which records are managed elsewhere and should not be modified or deleted.

Use case: Suppose a domain is managed by both DNSControl and a third-party system. This creates a problem because DNSControl will try to delete records inserted by the other system.  The other system may get confused and re-insert those records.  The two systems will get into an endless update cycle where each will revert changes made by the other in an endless loop.

To solve this problem simply include `IGNORE()` statements that identify which records are managed elsewhere.  DNSControl will not modify or delete those records.

Technically `IGNORE_NAME` is a promise that DNSControl will not modify or delete existing records that match particular patterns. It is like [`NO_PURGE`](../domain-modifiers/NO_PURGE.md) that matches only specific records.

Including a record that is ignored is considered an error and may have undefined behavior. This safety check can be disabled using the [`DISABLE_IGNORE_SAFETY_CHECK`](../domain-modifiers/DISABLE_IGNORE_SAFETY_CHECK.md) feature.

## Syntax

The `IGNORE()` function can be used with up to 3 parameters:

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  IGNORE(labelSpec, typeSpec, targetSpec),
  IGNORE(labelSpec, typeSpec),
  IGNORE(labelSpec),
);
```
{% endcode %}

* `labelSpec` is a glob that matches the DNS label. For example `"foo"` or `"foo*"`.  `"*"` matches all labels, as does the empty string (`""`).
* `typeSpec` is a comma-separated list of DNS types.  For example `"A"` matches DNS A records, `"A,CNAME"` matches both A and CNAME records. `"*"` matches any DNS type, as does the empty string (`""`).
* `targetSpec` is a glob that matches the DNS target. For example `"foo"` or `"foo*"`.  `"*"` matches all targets, as does the empty string (`""`).

`typeSpec` and `targetSpec` default to `"*"` if they are omitted.

## Globs

The `labelSpec` and `targetSpec` parameters supports glob patterns in the style of the [gobwas/glob](https://github.com/gobwas/glob) library.  All of the following patterns will work:

* `IGNORE("*.foo")` will ignore all records in the style of `bar.foo`, but will not ignore records using a double subdomain, such as `foo.bar.foo`.
* `IGNORE("**.foo")` will ignore all subdomains of `foo`, including double subdomains.
* `IGNORE("?oo")` will ignore all records of three symbols ending in `oo`, for example `foo` and `zoo`. It will not match `.`
* `IGNORE("[abc]oo")` will ignore records `aoo`, `boo` and `coo`. `IGNORE("[a-c]oo")` is equivalent.
* `IGNORE("[!abc]oo")` will ignore all three symbol records ending in `oo`, except for `aoo`, `boo`, `coo`.        `IGNORE("[!a-c]oo")` is equivalent.
* `IGNORE("{bar,[fz]oo}")` will ignore `bar`, `foo` and `zoo`.
* `IGNORE("\\*.foo")` will ignore the literal record `*.foo`.

## Typical Usage

General examples:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  IGNORE("foo"), // matches any records on foo.example.com
  IGNORE("baz", "A"), // matches any A records on label baz.example.com
  IGNORE("*", "MX", "*"), // matches all MX records
  IGNORE("*", "CNAME", "dev-*"), // matches CNAMEs with targets prefixed `dev-*`
  IGNORE("bar", "A,MX"), // ignore only A and MX records for name bar
  IGNORE("*", "*", "dev-*"), // Ignore targets with a `dev-` prefix
  IGNORE("*", "A", "1\.2\.3\."), // Ignore targets in the 1.2.3.0/24 CIDR block
);
```
{% endcode %}

Ignore Let's Encrypt (ACME) validation records:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  IGNORE("_acme-challenge", "TXT"),
  IGNORE("_acme-challenge.**", "TXT"),
);
```
{% endcode %}

Ignore DNS records typically inserted by Microsoft ActiveDirectory:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  IGNORE("_gc", "SRV"), // General Catalog
  IGNORE("_gc.**", "SRV"), // General Catalog
  IGNORE("_kerberos", "SRV"), // Kerb5 server
  IGNORE("_kerberos.**", "SRV"), // Kerb5 server
  IGNORE("_kpasswd", "SRV"), // Kpassword
  IGNORE("_kpasswd.**", "SRV"), // Kpassword
  IGNORE("_ldap", "SRV"), // LDAP
  IGNORE("_ldap.**", "SRV"), // LDAP
  IGNORE("_msdcs", "NS"), // Microsoft Domain Controller Service
  IGNORE("_msdcs.**", "NS"), // Microsoft Domain Controller Service
  IGNORE("_vlmcs", "SRV"), // FQDN of the KMS host
  IGNORE("_vlmcs.**", "SRV"), // FQDN of the KMS host
  IGNORE("domaindnszones", "A"),
  IGNORE("domaindnszones.**", "A"),
  IGNORE("forestdnszones", "A"),
  IGNORE("forestdnszones.**", "A"),
);
```
{% endcode %}

## Detailed examples

Here are some examples that illustrate how matching works.

All the examples assume the following DNS records are the "existing" records that a third-party is maintaining. (Don't be confused by the fact that we're using DNSControl notation for the records. Pretend some other system inserted them.)

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    A("@", "151.101.1.69"),
    A("www", "151.101.1.69"),
    A("foo", "1.1.1.1"),
    A("bar", "2.2.2.2"),
    CNAME("cshort", "www"),
    CNAME("cfull", "www.plts.org."),
    CNAME("cfull2", "www.bar.plts.org."),
    CNAME("cfull3", "bar.www.plts.org."),
);

D_EXTEND("more.example.com",
    A("foo", "1.1.1.1"),
    A("bar", "2.2.2.2"),
    CNAME("mshort", "www"),
    CNAME("mfull", "www.plts.org."),
    CNAME("mfull2", "www.bar.plts.org."),
    CNAME("mfull3", "bar.www.plts.org."),
);
```
{% endcode %}

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IGNORE("@", "", ""),
);
```
{% endcode %}

**Would match**:

* `foo.example.com. A 1.1.1.1`
* `foo.more.example.com. A 1.1.1.1`

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IGNORE("example.com.", "", ""),
);
```
{% endcode %}

**Would match**:

* nothing

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IGNORE("foo", "", ""),
);
```
{% endcode %}

**Would match**:

* `foo.example.com. A 1.1.1.1`

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IGNORE("foo.**", "", ""),
);
```
{% endcode %}

**Would match**:

* `foo.more.example.com. A 1.1.1.1`

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IGNORE("www", "", ""),
);
    //    www.example.com. A 174.136.107.196
```
{% endcode %}

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IGNORE("www.*", "", ""),
);
    //    nothing
```
{% endcode %}

**Would match**:

* nothing

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IGNORE("www.example.com", "", ""),
);
    //    nothing
```
{% endcode %}

**Would match**:

* nothing

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IGNORE("www.example.com.", "", ""),
);
```
{% endcode %}

**Would match**:

* none

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    //IGNORE("", "", "1.1.1.*"),
);
```
{% endcode %}

**Would match**:

* `foo.example.com. A 1.1.1.1`
* `foo.more.example.com. A 1.1.1.1`

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    //IGNORE("", "", "www"),
);
```
{% endcode %}

**Would match**:

* none

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IGNORE("", "", "*bar*"),
);
```
{% endcode %}

**Would match**:

* `cfull2.example.com. CNAME www.bar.plts.org.`
* `cfull3.example.com. CNAME bar.www.plts.org.`
* `mfull2.more.example.com. CNAME www.bar.plts.org.`
* `mfull3.more.example.com. CNAME bar.www.plts.org.`

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IGNORE("", "", "bar.**"),
);
```
{% endcode %}

**Would match**:

* `cfull3.example.com. CNAME bar.www.plts.org.`
* `mfull3.more.example.com. CNAME bar.www.plts.org.`

## Conflict handling

It is considered as an error for a `dnsconfig.js` to both ignore and insert the same record in a domain. This is done as a safety mechanism.

This will generate an error:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    ...
    TXT("myhost", "mytext"),
    IGNORE("myhost", "*", "*"),  // Error!  Ignoring an item we inserted
    ...
```
{% endcode %}

To disable this safety check, add the `DISABLE_IGNORE_SAFETY_CHECK` statement to the `D()`.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    DISABLE_IGNORE_SAFETY_CHECK,
    ...
    TXT("myhost", "mytext"),
    IGNORE("myhost", "*", "*"),
    ...
```
{% endcode %}

{% hint style="info" %}
FYI: Previously DNSControl permitted disabling this check on
a per-record basis using `IGNORE_NAME_DISABLE_SAFETY_CHECK`:
{% endhint %}

The `IGNORE_NAME_DISABLE_SAFETY_CHECK` feature does not exist in the diff2 world and its use will result in a validation error. Use the above example instead.

{% code %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    // THIS NO LONGER WORKS! Use DISABLE_IGNORE_SAFETY_CHECK instead. See above.
    TXT("myhost", "mytext", IGNORE_NAME_DISABLE_SAFETY_CHECK),
);
```
{% endcode %}

## Caveats

{% hint style="warning" %}
**WARNING**: Two systems updating the same domain is complex.  Complex things are risky. Use `IGNORE()`
as a last resort. Even then, test extensively.
{% endhint %}

* There is no locking.  If the external system and DNSControl make updates at the exact same time, the results are undefined.
* `IGNORE` works fine with records inserted into a `D()` via `D_EXTEND()`. The matching is done on the resulting FQDN of the label or target.
* `targetSpec` does not match fields other than the primary target.  For example, `MX` records have a target hostname plus a priority. There is no way to match the priority.
* The BIND provider can not ignore records it doesn't know about.  If it does not have access to an existing zonefile, it will create a zonefile from scratch. That new zonefile will not have any external records.  It will seem like they were not ignored, but in reality BIND didn't have visibility to them so that they could be ignored.
