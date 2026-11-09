---
name: MIGADU_BUILDER
parameters:
  - name
  - opts
  - modifiers...
parameter_types:
  name: string
  opts: "{ label?: string; verify?: string; mx?: boolean; wildcard?: boolean; dkim?: boolean; spf?: boolean; autoconfig?: boolean; autodiscover?: boolean; clientSRV?: boolean }"
  "modifiers...": RecordModifier[]
---

`MIGADU_BUILDER` creates the DNS records that [Migadu](https://www.migadu.com/)
requires for one mail domain: the two [`MX`](MX.md) records, the three DKIM
[`CNAME`](CNAME.md) records, and the client configuration records
(the Thunderbird-style `autoconfig` `CNAME`, the Outlook-style Autodiscover
[`SRV`](SRV.md) record, and the [RFC 6186](https://www.rfc-editor.org/rfc/rfc6186)
`SRV` records for submission, IMAP and POP3). On request it also creates the
domain verification [`TXT`](TXT.md) record, the SPF record, and a wildcard `MX`
for subdomain addressing.

The first parameter is the mail domain name. It must be the name the records
are created under, because the DKIM targets embed it.

`MIGADU_BUILDER` creates the records whose values Migadu fixes. It does not
create a DMARC record, because the policy is yours to choose; use
[`DMARC_BUILDER`](DMARC_BUILDER.md). See [SPF](#spf) for why the SPF record is
off by default.

## Simple example

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  MIGADU_BUILDER("example.com", {
      verify: "abcd1234",
      spf: true,
  }),
);
```
{% endcode %}

This yields the following records:

```text
@                   IN  TXT    "hosted-email-verify=abcd1234"
@                   IN  MX     10 aspmx1.migadu.com.
@                   IN  MX     20 aspmx2.migadu.com.
key1._domainkey     IN  CNAME  key1.example.com._domainkey.migadu.com.
key2._domainkey     IN  CNAME  key2.example.com._domainkey.migadu.com.
key3._domainkey     IN  CNAME  key3.example.com._domainkey.migadu.com.
@                   IN  TXT    "v=spf1 include:spf.migadu.com -all"
autoconfig          IN  CNAME  autoconfig.migadu.com.
_autodiscover._tcp  IN  SRV    0 1 443 autodiscover.migadu.com.
_submissions._tcp   IN  SRV    0 1 465 smtp.migadu.com.
_imaps._tcp         IN  SRV    0 1 993 imap.migadu.com.
_pop3s._tcp         IN  SRV    0 1 995 pop.migadu.com.
```

## Parameters

| Option | Type | Default | Description |
|---|---|---|---|
| `label` | string | `"@"` | The label of the mail domain within the zone. Applies to every record the builder creates. |
| `verify` | string | none | The verification token from the Migadu admin panel, for example `"abcd1234"`. Creates the `TXT` record `hosted-email-verify=<token>`. Give the token only, without the `hosted-email-verify=` prefix. |
| `mx` | boolean | `true` | Create the two `MX` records. |
| `wildcard` | boolean | `false` | Also create the two `MX` records on the wildcard below the mail domain, for [subdomain addressing](https://migadu.com/guides/subdomain_addressing/). Requires `mx`. |
| `dkim` | boolean | `true` | Create the three DKIM `CNAME` records. |
| `spf` | boolean | `false` | Create the SPF `TXT` record `v=spf1 include:spf.migadu.com -all`. See [SPF](#spf). |
| `autoconfig` | boolean | `true` | Create the `autoconfig` `CNAME` record used by Thunderbird and other clients that follow the Mozilla convention. |
| `autodiscover` | boolean | `true` | Create the `_autodiscover._tcp` `SRV` record used by Outlook. |
| `clientSRV` | boolean | `true` | Create the RFC 6186 `SRV` records `_submissions._tcp`, `_imaps._tcp` and `_pop3s._tcp`. |

Unknown options and values of the wrong type are errors.

## Finding your values

Only the verification token differs between domains. It is shown in the Migadu
admin panel under **Domains → your domain → DNS**. Compare the records the
builder creates with that page before switching the `MX` records over.

## Subdomains

A mail domain that is a subdomain of the zone is placed with `label`. The first
parameter stays the full mail domain name, because the DKIM targets embed it.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  MIGADU_BUILDER("mail.example.com", {
      label: "mail",
      verify: "abcd1234",
      wildcard: true,
      autoconfig: false,
      autodiscover: false,
      clientSRV: false,
  }),
);
```
{% endcode %}

This yields the following records:

```text
mail                  IN  TXT    "hosted-email-verify=abcd1234"
mail                  IN  MX     10 aspmx1.migadu.com.
mail                  IN  MX     20 aspmx2.migadu.com.
*.mail                IN  MX     10 aspmx1.migadu.com.
*.mail                IN  MX     20 aspmx2.migadu.com.
key1._domainkey.mail  IN  CNAME  key1.mail.example.com._domainkey.migadu.com.
key2._domainkey.mail  IN  CNAME  key2.mail.example.com._domainkey.migadu.com.
key3._domainkey.mail  IN  CNAME  key3.mail.example.com._domainkey.migadu.com.
```

Inside [`D_EXTEND`](../top-level-functions/D_EXTEND.md), the records are created below the
subdomain, so the first parameter is the subdomain's full name and `label` is
not needed:

{% code title="dnsconfig.js" %}
```javascript
D_EXTEND("mail.example.com",
  MIGADU_BUILDER("mail.example.com", {
      verify: "abcd1234",
  }),
);
```
{% endcode %}

## SPF

A name may have only one SPF record. Two SPF records at the same name are a
`PermError` under [RFC 7208, section 4.5](https://www.rfc-editor.org/rfc/rfc7208#section-4.5),
not a merge, so a domain that also sends mail through another service breaks as
soon as the builder adds its own. Set `spf: true` only when Migadu is the only
service that sends mail for the domain. Otherwise leave it off and add
`include:spf.migadu.com` to [`SPF_BUILDER`](SPF_BUILDER.md):

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  MIGADU_BUILDER("example.com", {
      verify: "abcd1234",
  }),
  SPF_BUILDER({
      label: "@",
      parts: [
          "v=spf1",
          "include:spf.migadu.com",
          "include:_spf.example.net",
          "-all",
      ],
  }),
);
```
{% endcode %}

## TTL

The TTL is set with [`TTL`](../record-modifiers/TTL.md) as a further argument to
the call. It applies to every record the builder creates, and it is the only
record modifier the call takes. Without a `TTL()` the zone's
[`DefaultTTL`](DefaultTTL.md) applies.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  MIGADU_BUILDER("example.com", {
      verify: "abcd1234",
  }, TTL("1h")),
);
```
{% endcode %}

## References

- [Subdomain addressing](https://migadu.com/guides/subdomain_addressing/).
  The wildcard `MX` records.
- [RFC 6186: Use of SRV Records for Locating Email Submission/Access Services](https://www.rfc-editor.org/rfc/rfc6186)
  and [RFC 8314: Cleartext Considered Obsolete](https://www.rfc-editor.org/rfc/rfc8314).
  The `_submissions._tcp`, `_imaps._tcp` and `_pop3s._tcp` records.
- [RFC 7208, section 4.5](https://www.rfc-editor.org/rfc/rfc7208#section-4.5).
  Why a name may have only one SPF record.
