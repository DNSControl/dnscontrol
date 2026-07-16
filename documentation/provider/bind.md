This provider maintains a directory with a collection of .zone files as appropriate for ISC BIND, and other systems that use the RFC 1035 zone-file format.

This provider does not generate or update the named.conf file, nor does it deploy the .zone files to the BIND master. Both of those tasks are different at each site, so they are best done by a locally-written script.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `BIND`.

Optional fields include:

* `directory`: Location of the zone files.  Default: `zones` (in the current directory).
* [`filenameformat`](#filenameformat): The formula used to generate the zone filenames. The default is usually sufficient.  Default: `"%c.zone"`

Example:

{% code title="creds.json" %}
```json
{
  "bind": {
    "TYPE": "BIND",
    "directory": "myzones"
  }
}
```
{% endcode %}

As of v4.2.0 `dnscontrol push` will create subdirectories along the path to the filename. This includes both the portion of the path created by the `directory` setting and the `filenameformat` setting. For security reasons, the automatic creation of subdirectories is disabled if `dnscontrol` is running as root. (Running DNSControl as root is not recommended in general.)

## Meta configuration

This provider accepts some optional metadata in the `NewDnsProvider()` call.

* `default_soa`: If no SOA record exists in a zone file, one will be created based on the values specified here. Use `SOA()` to update existing zone files.
* `default_ns`: Inject these NS records into the zone.  Use this when `NS()` is insufficient.

In this example we set the default SOA settings and NS records.

{% code title="dnsconfig.js" %}
```javascript
var DSP_BIND = NewDnsProvider("bind", {
    "default_soa": {
        "master": "ns1.example.com.",
        "mbox": "spamtrap.example.com.",
        "refresh": 3600,
        "retry": 600,
        "expire": 604800,
        "minttl": 1440,
    },
    "default_ns": [
        "ns1.example.com.",
        "ns2.example.com.",
        "ns3.example.com.",
        "ns4.example.com."
    ]
})
```
{% endcode %}

# FYI: SOA Records

SOA records are a bit weird in DNSControl.   Most providers auto-generate SOA records and do not permit any modifications. BIND is unique in that it requires users to manage the SOA records themselves.

Because BIND is unique, BIND's SOA support is kind of a hack.  It leaves the SOA record alone, with 2 exceptions:

1. The serial number: If something in the zone changes, the serial number is incremented (see below).
2. Missing SOAs: If there is no SOA record in a zone (or the zone is being created for the first time), the SOA is created.  The initial values are taken from the `default_soa` settings.

The `default_soa` values are only used when creating an SOA for the first time. The values are not used to update an SOA.  Most people edit the SOA values by manually editing the zonefile or using the `SOA()` function.

# FYI: SOA serial numbers

DNSControl maintains beautiful zone serial numbers.

DNSControl tries to maintain the serial number as yyyymmddvv. The algorithm for increasing the serial number is to select the max of (current serial + 1) and (yyyymmdd00). If you use a number larger than today's date (say, 2099000099) DNSControl will simply increment it forever.

The good news is that DNSControl is smart enough to only increment a zone's serial number if something in the zone changed. It does not increment the serial number just because DNSControl ran.

DNSControl does not handle special serial number math such as "looping through zero" nor does it pay attention to the rules around the maximum delta permitted. Those are simply avoided because yyyymmdd99 fits in the first quadrant of the 32-bit serial number space. If you don't understand this paragraph consider yourself lucky; with DNSControl you don't need to.

# filenameformat

The `filenameformat` parameter specifies the file name to be used when writing the zone file. The default (`%c.zone`) is acceptable in most cases: the file name is the name as specified in the `D()` function plus ".zone".

The filenameformat is a string with a few printf-like `%` verbs:

| Verb    | Description                                       | `EXAMple.com` | `EXAMple.com!MyTag` | `рф.com!myTag`       |
| ------- | ------------------------------------------------- | ------------- | ------------------- | -------------------- |
| `%T`    | the tag                                           | "" (null)     | `myTag`             | `myTag`              |
| `%c`    | canonical name, globally unique and comparable    | `example.com` | `example.com!myTag` | `xn--p1ai.com!myTag` |
| `%a`    | ASCII domain (Punycode, downcased)                | `example.com` | `example.com`       | `xn--p1ai.com`       |
| `%u`    | Unicode domain (non-Unicode parts downcased)      | `example.com` | `example.com`       | `рф.com`             |
| `%r`    | Raw (unmodified) Domain from `D()` (risky!)       | `EXAMple.com` | `EXAMple.com`       | `рф.com`             |
| `%f`    | like `%c` but better for filenames (`%a%?_%T`)    | `example.com` | `example.com_myTag` | `xn--p1ai.com_myTag` |
| `%F`    | like `%f` but reversed order (`%T%?_%a`)          | `example.com` | `myTag_example.com` | `myTag_xn--p1ai.com` |
| `%?x`   | returns `x` if tag exists, otherwise ""           | "" (null)     | `x`                 | `x`                  |
| `%%`    | a literal percent sign                            | `%`           | `%`                 | `%`                  |
| `a-Z./` | other printable characters are copied exactly     | `a-Z./`       | `a-Z./`             | `a-Z./`              |
| `%U`    | (deprecated, use `%c`) Same as `%D%?!%T` (risky!) | `example.com` | `example.com!myTag` | `рф.com!myTag`       |
| `%D`    | (deprecated, use `%r`) mangles Unicode (risky!)   | `example.com` | `example.com`       | `рф.com`             |

* `%?x` is typically used to generate an optional `!` or `_` if there is a tag.
* `%r` is considered "risky" because it can produce a domain name that is not canonical. For example, if you use `D("FOO.com")` and later change it to `D("foo.com")`, your file names will change.
* Format strings must not end with an incomplete `%` or `%?`
* Generating a filename without a tag is risky. For example, if the same `dnsconfig.js` has `D("example.com!inside", DSP_BIND)` and `D("example.com!outside", DSP_BIND)`, both will use the same filename. DNSControl will write both zone files to the same file, flapping between the two. No error or warning will be output.

Useful examples:

| Verb         | Description                         | `EXAMple.com`      | `EXAMple.com!MyTag`      | `рф.com!myTag`            |
| ------------ | ----------------------------------- | ------------------ | ------------------------ | ------------------------- |
| `%c.zone`    | Default format (v4.28 and later)    | `example.com.zone` | `example.com!myTag.zone` | `xn--p1ai.com!myTag.zone` |
| `%U.zone`    | Default format (pre-v4.28) (risky!) | `example.com.zone` | `example.com!myTag.zone` | `рф.com!myTag.zone`       |
| `db_%f`      | Recommended in a popular DNS book   | `db_example.com`   | `db_example.com_myTag`   | `db_xn--p1ai.com_myTag`   |
| `db_%a%?_%T` | same as above but using `%?_`       | `db_example.com`   | `db_example.com_myTag`   | `db_xn--p1ai.com_myTag`   |

Compatibility notes:

* `%D` should not be used. It downcases the string in a way that is probably incompatible with Unicode characters.  It is retained for compatibility with pre-v4.28 releases. If your domain has capital Unicode characters, backwards compatibility is not guaranteed. Use `%r` instead.
* `%U` relies on `%D` which is deprecated. Use `%c` instead.
* As of v4.28 the default format string changed from `%U.zone` to `%c.zone`. This should only matter if your `D()` statements included non-ASCII (Unicode) runes that were capitalized.
* If you are using pre-v4.28 releases the above table is slightly misleading because uppercase ASCII letters do not always work. If you are using pre-v4.28 releases, assume the above table lists `example.com` instead of `EXAMpl.com`.

# FYI: get-zones

The DNSControl `get-zones all` subcommand scans the directory for any files named `*.zone` and assumes they are zone files.

```shell
dnscontrol get-zones --format=nameonly - BIND all
```

If `filenameformat` is defined, `dnscontrol` makes a guess at which filenames are zones by reversing the logic of the format string. It doesn't try very hard to get this right, as getting it right in all situations is mathematically impossible.  Feel free to file an issue if find a situation where it doesn't work. I love a challenge!

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ✅
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❔
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ✅
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ✅
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ✅
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
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ✅
  - [`DS`](../language-reference/domain-modifiers/DS.md): ✅
<!-- provider-features-end -->
