---
name: TXT
parameters:
  - name
  - contents
  - modifiers...
parameter_types:
  name: string
  contents: string | string[]
  "modifiers...": RecordModifier[]
---

`TXT` adds a [Text record](https://www.rfc-editor.org/rfc/rfc1035) to a domain. The name should be the relative label for the record. Use `@` for the domain apex.

The contents is either a single or multiple strings. To specify multiple strings, specify them as an array.

Each string is a JavaScript string (quoted using single or double quotes). The (somewhat complex) quoting rules of the DNS protocol will be done for you.

Modifiers can be any number of [record modifiers](https://docs.dnscontrol.org/language-reference/record-modifiers) or JSON objects, which will be merged into the record's metadata.

{% code title="dnsconfig.js" %}
```javascript
    D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
      TXT("@", "598611146-3338560"),
      TXT("listserve", "google-site-verification=12345"),
      TXT("multiple", ["one", "two", "three"]),  // Multiple strings
      TXT("quoted", 'any "quotes" and escapes? ugh; no worries!'),
      TXT("_domainkey", "t=y; o=-;"), // Escapes are done for you automatically.
      TXT("long", "X".repeat(300)), // Long strings are split automatically.
    );
```
{% endcode %}

{% hint style="info" %}
**NOTE**: In the past, long strings had to be annotated with the keyword
`AUTOSPLIT`. This is no longer required. The keyword is now a no-op.
{% endhint %}

### Long strings

Strings that are longer than 255 octets (bytes) will be quietly split into 255-octets chunks or the provider may report an error if it does not handle multiple strings.

### TXT record edge cases

Most providers do not support the full possibilities of what a `TXT` record can store. DNSControl can not handle all the edge cases and incompatibles that providers have introduced. Instead, it stores the string(s) that you provide and passes them to the provider verbatim. The provider may opt to accept the data, fix it, or reject it. This happens early in the processing, long before the DNSControl talks to the provider's API.

The RFCs specify that a `TXT` record stores one or more strings, each is up to 255 octets (bytes) long. We call these individual strings *chunks*. Each chunk may be zero to 255 octets long. There is no limit to the number of chunks in a `TXT` record, other than IP packet length restrictions. The contents of each chunk may be octets of value from 0x00 to 0xff.

In reality DNS Service Providers (DSPs) place many restrictions on `TXT` records.

Some DSPs only support a single string of 255 octets or fewer. Multiple strings, or any one string being longer than 255 octets will result in an error. One provider limits the string to 254 octets, which makes me think they're code has an off-by-one error.

Some DSPs only support one string, but it may be of any length. Behind the scenes the provider splits it into 255-octet chunks (except the last one, of course).

Some DSPs support multiple strings, but API requests must be 512-bytes or fewer, and with quoting, escaping, and other encoding mishegoss you can't be sure what will be permitted until you actually try it.

Regardless of the quantity and length of strings, some providers ban double quotes, back-ticks, or other chars.

### Testing the support of a provider

#### How can you tell if a provider will support a particular `TXT()` record?

Include the `TXT()` record in a [`D()`](../top-level-functions/D.md) as usual, along with the `DnsProvider()` for that provider. Run `dnscontrol check` to see if any errors are produced. The check command does not talk to the provider's API, thus permitting you to do this without having an account at that provider.

#### What if the provider rejects a string that is supported?

Suppose I can create the TXT record using the DSP's web portal but DNSControl rejects the string?

It is possible that the provider code in DNSControl rejects strings that the DSP accepts. This is because the test is done in code, not by querying the provider's API. It is possible that the code was written to work around a bug (such as rejecting a string with a back-tick) but now that bug has been fixed.

All such checks are in `providers/${providername}/auditrecords.go`. You can try removing the check that you feel is in error and see if the provider's API accepts the record. You can do this by running the integration tests, or by simply adding that record to an existing `dnsconfig.js` and seeing if `dnscontrol push` is able to push that record into production. (Be careful if you are testing this on a domain used in production.)
