---
name: NAPTR
parameters:
  - subdomain
  - order
  - preference
  - terminalflag
  - service
  - regexp
  - target
  - modifiers...
parameter_types:
  subdomain: string
  order: number
  preference: number
  terminalflag: string
  service: string
  regexp: string
  target: string
  "modifiers...": RecordModifier[]
---

## Introduction

NAPTR adds a NAPTR record to the domain. Various formats exist. NAPTR is a part of DDDS such as ENUM (defined by [RFC 6116](https://www.rfc-editor.org/rfc/rfc6116)), SIP ([RFC 3263](https://www.rfc-editor.org/rfc/rfc3263)), S-NAPTR ([RFC 3958](https://www.rfc-editor.org/rfc/rfc3958)) or U-NAPTR ([RFC 4848](https://www.rfc-editor.org/rfc/rfc4848)).

## Parameters

### `subdomain`

Subdomain of the domain (e.g. `example.com`) this entry represents.

#### E164
In the case of E164 (e.g. `3.2.1.5.5.5.0.0.8.1.e164.arpa.`) - where [`terminalflag`](#terminalflag) is `u` - the final digit of the zone it represents, or the zone apex record `@`. For example, the ARPA zone `3.2.1.5.5.5.0.0.8.1.e164.arpa.` represents the phone number block 001800555123*X* (or the synonymous +1800555123*X*), where *X* is the final digit of the phone number string, i.e. the [`subdomain`](#subdomain).


### `order`

ordinal (1st, 2nd, 3rd, ...) 16 bit number (2^16 i.e. <= 65535) which determines lower entries are sent first (`1`), and  higher, last (`65535`).

### `preference`

16 bit number (2^16 i.e. <= 65535). At the DNS server, this entry is summed with other entries of identical [`order`](#order) value and normalised to a fraction of 100 percent, determining the likelihood that this record is returned by the DNS system. Effective for load balancing services.

### `terminalflag`
(case insensitive)

One of [AaSsUuPp], where:
 * `a` (terminal lookup) means that the output of the [`target`](#target) rewrite will be a domain-name for which an [`A`](A.md) or [`AAAA`](AAAA.md) record should be queried
 * `p` Protocol specific
 * `s` (terminal lookup) indicates that [`target`](#target) points to a [`SRV`](SRV.md) record
 * `u` (terminal lookup) indicates that [`target`](#target) is a (SIP) URN or URI
 * "" (empty string) - a non-terminal condition defined by the ENUM application ([RFC 6116](https://www.rfc-editor.org/rfc/rfc6116)) to indicate that regexp is empty and the replace field contains the FQDN of another NAPTR RR


Mutually exclusive; more than one cannot be combined in the same record. Since there is no place for a port specification in the NAPTR record, when the `a` [`terminalflag`](#terminalflag) is used, the specified protocol must be running on its default port (Note that at least SIP URI forms allow ports to be specified).

Flags called 'terminal' halt the looping rewrite algorithm of DNS.


### `service`
(case insensitive)

*`protocol+rs`* where *`protocol`* defines the protocol used by the DDDS application. *`rs`* is the resolution service. There may be 0 or more resolution services each separated by `+`. ENUM further defines this to be a type field and allows a subtype separated by a colon (`:`).

For E164, typically one of `E2U+SIP` (or `E2U+sip`) or `E2U+email`. For SIP, typically `SIPS+D2T` for TCP/TLS `sips:` URIs, or TLS `sip:` URIs, or `SIP+D2T` for TCP based SIP, or `SIP+D2U` for UDP based SIP. Note that SCTP, WS and WSS are also available.


Valid [IANA registered services for ENUM](https://www.iana.org/assignments/enum-services/enum-services.xhtml#enum-services-1):
```text
E2U+pres
E2U+voice:tel+sms:tel (compound form)
E2U+web:http
E2U+sms:mailto
E2U+sms:tel
E2U+sip
E2U+pstn
E2U+tel
```

Valid [IANA registered SIP services](https://www.iana.org/assignments/sip-table/sip-table.xhtml#sip-table-1):

```text
SIP+D2T
SIPS+D2T
SIP+D2U
SIP+D2S
SIPS+D2S
SIP+D2W
SIPS+D2W
```

### `regexp`

[Syntax: `delimit ere delimit substitution delimit flag`] an ERE or extended regular expression which captures any address string `.*` found between the line start `^` and finish `$` anchors (i.e. `!^.*$!`), and redirects it to the stated `sip:`, `sips:`, `tel:` or `mailto:` URI. Other URI forms may be possible. Other delimiter (`!`) forms are possible. The final `flag`, if any, shall be `i`, i.e. case **i**nsensitive.

Examples (taken from [Zytrax](https://www.zytrax.com/books/dns/ch8/naptr.html#regex-examples)):
```text
# AUS = Application User String
# all examples use ! as the delimiter for consistency
# and simplicity
# AUS = +441115551234 in all cases

!(\\+441115551234)!tel:\\1!
# explicit check of all characters in string
# the +441115551234 because of () creates a group
# which is referenced by \1 in substitution
# result = tel:+441115551234

!^(\\+441115551234)$!tel:\\1!
# this is functionally identical to the expression
# above but uses ^ and $ to anchor both ends of
# the expression, there is no technical reason to do this
# within an ere and the RFCs are silent on the topic
# result = tel:+441115551234

!(.+)!tel:\\1!
# given the AUS of +441115551234
# the expression (.+) sets back ref 1 = +441115551234
# . = any character, + = 0 or more times
# result = tel:+441115551212

!\\+44111(.+)!sip:775\\1@some.example.com!
# given the AUS of +441115551234 provides partial replacement
# removes the 44111 part and substitutes 775
# result = sip:7755551234@some.example.com

!.*!sip:james@sip.example.com!
# reads and ignores AUS using .*
# and is called a simple replacement expression
# result = sip:james@sip.example.com
```

U-NAPTR supported regexp fields must be of the form (from the RFC):

```text
"!.*!<URI>!"
# the .* (any character 1 or more times)
# is fixed by the RFC and essentially ignores
# the AUS data. The result will always be URI
```


### `target`

A (replacement) record for the target - format depends on [`terminalflag`](#terminalflag).
 * A [`SRV`](SRV.md), if the [`terminalflag`](#terminalflag) is `s` (syntax: *`_Service._Proto.Name`*)
 * An [`A`](A.md) or [`AAAA`](AAAA.md) if the [`terminalflag`](#terminalflag) is `a`
 * URI if the [`terminalflag`](#terminalflag) is `u`


Not all examples are guaranteed to be standards compliant, or correct.

## Examples

### Examples for e164 ARPA:

Individual e164 records

{% code title="dnsconfig.js" %}
```javascript
D("3.2.1.5.5.5.0.0.8.1.e164.arpa.", REG_MY_PROVIDER, DnsProvider(R53),
  NAPTR("1",  10, 10, "u", "E2U+SIP", "!^.*$!sip:bob@example.com!", "."),
  NAPTR("2",  10, 10, "u", "E2U+SIP", "!^.*$!sip:alice@example.com!", "."),
  NAPTR("4",  10, 10, "u", "E2U+SIP", "!^.*$!sip:kate@example.com!", "."),
  NAPTR("5",  10, 10, "u", "E2U+SIP", "!^.*$!sip:steve@example.com!", "."),
  NAPTR("6",  10, 10, "u", "E2U+SIP", "!^.*$!sip:joe@example.com!", "."),
  NAPTR("7",  10, 10, "u", "E2U+SIP", "!^.*$!sip:jane@example.com!", "."),
  NAPTR("8",  10, 10, "u", "E2U+SIP", "!^.*$!sip:mike@example.com!", "."),
  NAPTR("9",  10, 10, "u", "E2U+SIP", "!^.*$!sip:linda@example.com!", "."),
  NAPTR("0",  10, 10, "u", "E2U+SIP", "!^.*$!sip:fax@example.com!", "."),
);
```
{% endcode %}

Single e164 zone
{% code title="dnsconfig.js" %}
```javascript
D("4.3.2.1.5.5.5.0.0.8.1.e164.arpa.", REG_MY_PROVIDER, DnsProvider(R53),
  NAPTR("@", 100, 50, "u", "E2U+SIP", "!^.*$!sip:customer-service@example.com!", "."),
  NAPTR("@", 101, 50, "u", "E2U+email", "!^.*$!mailto:information@example.com!", "."),
  NAPTR("@", 101, 50, "u", "smtp+E2U", "!^.*$!mailto:information@example.com!", "."),
);
```
{% endcode %}


### Examples for SIP:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  NAPTR("@", 20, 50, "s", "SIPS+D2T", "", "_sips._tcp.example.com."),
  NAPTR("@", 20, 50, "s", "SIP+D2T", "", "_sip._tcp.example.com."),
  NAPTR("@", 30, 50, "s", "SIP+D2U", "", "_sip._udp.example.com."),
  NAPTR("help", 100, 50, "s", "SIP+D2U", "!^.*$!sip:customer-service@example.com!", "_sip._udp.example.com."),
  NAPTR("help", 101, 50, "s", "SIP+D2T", "!^.*$!sip:customer-service@example.com!", "_sip._tcp.example.com."),
  SRV("_sip._udp", 100, 0, 5060, "sip.example.com."),
  SRV("_sip._tcp", 100, 0, 5060, "sip.example.com."),
  SRV("_sips._tcp", 100, 0, 5061, "sip.example.com."),
  A("sip", "192.0.2.2"),
  AAAA("sip", "2001:db8::85a3"),
  // and so on
);
```
{% endcode %}


### Other RFC based examples:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  NAPTR("@",100, 50, "a", "z3950+N2L+N2C", "", "cidserver.example.com."),
  NAPTR("@", 50, 50, "a", "rcds+N2C", "", "cidserver.example.com."),
  NAPTR("@", 30, 50, "s", "http+N2L+N2C+N2R", "", "www.example.com."),
  NAPTR("www",100,100, "s", "http+I2R", "", "_http._tcp.example.com."),
  NAPTR("www",100,100, "s", "ftp+I2R", "", "_ftp._tcp.example.com."),
  SRV("_z3950._tcp", 0, 0, 1000, "z3950.beast.example.com."),
  SRV("_http._tcp", 10, 0, 80, "foo.example.com."),
  // and so on
);
```
{% endcode %}

