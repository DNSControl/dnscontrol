---
name: M365_BUILDER
parameters:
  - name
  - opts
  - modifiers...
parameter_types:
  name: string
  opts: "{ label?: string; mx?: boolean; mxPriority?: number; mxTarget?: string; autodiscover?: boolean; autodiscoverTarget?: string; dkim?: boolean; initialDomain?: string; dkimSelector1Target?: string; dkimSelector2Target?: string; skypeForBusiness?: boolean; sipFederationTarget?: string; mdm?: boolean; mdmEnrollmentTarget?: string; mdmRegistrationTarget?: string; domainGUID?: string; verificationToken?: string }"
  "modifiers...": RecordModifier[]
---

`M365_BUILDER` creates the DNS records that Microsoft 365 requires for one
domain: the [`MX`](MX.md) record for Exchange Online, the Autodiscover
[`CNAME`](CNAME.md), and the two DKIM `CNAME` records. On request it also
creates the domain verification [`TXT`](TXT.md) record, the SIP federation
[`SRV`](SRV.md) record for Microsoft Teams (historically Skype for Business),
and the two `CNAME` records for Microsoft Intune and Microsoft Entra device
registration.

The first parameter is the Microsoft 365 domain name. It must be the name the
records are created under, because the domain token that appears in the `MX` and
DKIM targets is derived from it. That is only checked while a target is actually
derived from it: a call that sets `domainGUID`, or that needs neither a derived
`MX` target nor derived DKIM targets, takes any name.

`M365_BUILDER` does not create `SPF` or `DMARC` records. A domain may only have
one SPF record, so it cannot be built in isolation from the rest of the domain's
mail senders. See [`SPF_BUILDER`](SPF_BUILDER.md) and
[`DMARC_BUILDER`](DMARC_BUILDER.md).

## Simple example

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  M365_BUILDER("example.com", {
      initialDomain: "contoso.onmicrosoft.com",
  }, TTL("1h")),
);
```
{% endcode %}

This yields the following records:

```text
@                     IN  MX     0 example-com.mail.protection.outlook.com.
autodiscover          IN  CNAME  autodiscover.outlook.com.
selector1._domainkey  IN  CNAME  selector1-example-com._domainkey.contoso.onmicrosoft.com.
selector2._domainkey  IN  CNAME  selector2-example-com._domainkey.contoso.onmicrosoft.com.
```

## Parameters

Every record has a switch that turns it on or off, and exactly one option that
replaces its target completely. Only the values that Microsoft documents as
derivable are derived; every value that Microsoft assigns can be given
literally. An option that belongs to a disabled record is still checked for its
type and its format; it just does not produce a record.

| Option | Type | Default | Description |
|---|---|---|---|
| `label` | string | `"@"` | The label of the Microsoft 365 domain within the zone. Applies to every record the builder creates. |
| `mx` | boolean | `true` | Create the `MX` record. |
| `mxPriority` | number | `0` | The preference of the `MX` record (0-65535). |
| `autodiscover` | boolean | `true` | Create the Autodiscover `CNAME` record. |
| `dkim` | boolean | `true` | Create both DKIM `CNAME` records. |
| `skypeForBusiness` | boolean | `false` | Create the SIP federation `SRV` record. |
| `mdm` | boolean | `false` | Create the Intune and Entra device registration `CNAME` records. |
| `verificationToken` | string | none | The full value of the domain verification `TXT` record, including the `MS=` prefix, for example `"MS=ms12345678"`. |
| `domainGUID` | string | derived | The token Microsoft assigned to this domain; a single label without dots. Derived from the domain name by replacing every `.` with `-`; there is no default for a domain name that contains a dash, which includes every internationalized name, because its punycode form begins with `xn--`. |
| `initialDomain` | string | none | The initial domain of the tenant, for example `"contoso.onmicrosoft.com"`. Used to derive the DKIM targets. |
| `mxTarget` | string | `<domainGUID>.mail.protection.outlook.com.` | The target of the `MX` record. A domain that Microsoft provisioned with an `mx.microsoft` target needs it set explicitly. |
| `autodiscoverTarget` | string | `autodiscover.outlook.com.` | The target of the Autodiscover `CNAME` record. |
| `dkimSelector1Target` | string | derived | The target of the `selector1._domainkey` record. |
| `dkimSelector2Target` | string | derived | The target of the `selector2._domainkey` record. |
| `sipFederationTarget` | string | `sipfed.online.lync.com.` | The target of the `_sipfederationtls._tcp` record. |
| `mdmEnrollmentTarget` | string | `enterpriseenrollment-s.manage.microsoft.com.` | The target of the `enterpriseenrollment` record. |
| `mdmRegistrationTarget` | string | `enterpriseregistration.windows.net.` | The target of the `enterpriseregistration` record. |

Target options and `initialDomain` may be written with or without a trailing
dot.

## DKIM

There are two ways to tell the builder where the DKIM records point.

The exact way works for every domain: set `dkimSelector1Target` and
`dkimSelector2Target` to the values Microsoft publishes for the domain. Both
must be set together. Query them with:

```text
Get-DkimSigningConfig -Identity example.com | Format-List Selector1CNAME,Selector2CNAME
```

The derived way is a convenience for the legacy target format: set
`initialDomain`, and the targets become
`selector1-<domainGUID>._domainkey.<initialDomain>.` and
`selector2-<domainGUID>._domainkey.<initialDomain>.`.

Domains that Microsoft created from May 2025 onwards get a target below
`dkim.mail.microsoft` instead, for example
`selector1-example-com._domainkey.contoso.n-v1.dkim.mail.microsoft`. That target
contains a partition character that Microsoft assigns and that is not
configurable, so it cannot be derived. If the Microsoft 365 admin center or
`Get-DkimSigningConfig` shows a target below `dkim.mail.microsoft`, copy both
values into `dkimSelector1Target` and `dkimSelector2Target`; `initialDomain` is
then not needed.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  M365_BUILDER("example.com", {
      dkimSelector1Target: "selector1-example-com._domainkey.contoso.n-v1.dkim.mail.microsoft",
      dkimSelector2Target: "selector2-example-com._domainkey.contoso.n-v1.dkim.mail.microsoft",
  }),
);
```
{% endcode %}

## Subdomains

A Microsoft 365 domain that is a subdomain of the zone is placed with `label`.
The first parameter stays the full Microsoft 365 domain name, because the domain
token is derived from it.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  M365_BUILDER("test.example.com", {
      label: "test",
      initialDomain: "contoso.onmicrosoft.com",
      verificationToken: "MS=ms12345678",
      skypeForBusiness: true,
      mdm: true,
  }, TTL("1h")),
);
```
{% endcode %}

This yields the following records:

```text
test                         IN  TXT    "MS=ms12345678"
test                         IN  MX     0 test-example-com.mail.protection.outlook.com.
autodiscover.test            IN  CNAME  autodiscover.outlook.com.
selector1._domainkey.test    IN  CNAME  selector1-test-example-com._domainkey.contoso.onmicrosoft.com.
selector2._domainkey.test    IN  CNAME  selector2-test-example-com._domainkey.contoso.onmicrosoft.com.
_sipfederationtls._tcp.test  IN  SRV    100 1 5061 sipfed.online.lync.com.
enterpriseregistration.test  IN  CNAME  enterpriseregistration.windows.net.
enterpriseenrollment.test    IN  CNAME  enterpriseenrollment-s.manage.microsoft.com.
```

Inside a [`D_EXTEND`](../top-level-functions/D_EXTEND.md) the records are placed
below the extended name, so the first parameter is the name of the extended
domain and `label` is not needed.

{% code title="dnsconfig.js" %}
```javascript
D_EXTEND("sub.example.com",
  M365_BUILDER("sub.example.com", {
      initialDomain: "contoso.onmicrosoft.com",
  }),
);
```
{% endcode %}

## DANE and DNSSEC for Exchange Online

A tenant that has been switched to DANE and DNSSEC gets a new, opaque `MX`
target from `Enable-DnssecForVerifiedDomain` (the `DnssecMxValue` field), and a
DKIM target in the new format. All three are set explicitly:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  M365_BUILDER("example.com", {
      mxTarget: "example-com.o-v1.mx.microsoft",
      dkimSelector1Target: "selector1-example-com._domainkey.contoso.o-v1.dkim.mail.microsoft",
      dkimSelector2Target: "selector2-example-com._domainkey.contoso.o-v1.dkim.mail.microsoft",
  }),
  // Microsoft's procedure keeps the legacy record until the new one has been
  // verified: publish the new target with mxPriority 20 first, then give it
  // the preference 0 used here, move the legacy record to 30, and delete it.
  MX("@", 30, "example-com.mail.protection.outlook.com."),
);
```
{% endcode %}

The matching `TLSA` records live in Microsoft's zone, not in yours, so the
builder does not create any. Use [`TLSA`](TLSA.md) if you need one for another
reason.

A domain added as an accepted domain from 1 July 2026 onwards is provisioned
with an `mx.microsoft` target from the start, without DANE having been requested
for it. The builder keeps deriving the `mail.protection.outlook.com` target,
because that is the one an existing domain keeps; set `mxTarget` to the value
the Microsoft 365 admin center shows for the domain.

## Sovereign clouds (GCC High, DoD, 21Vianet)

The sovereign clouds use different targets. There is no cloud profile; set the
target options instead.

| | GCC High | DoD | 21Vianet |
|---|---|---|---|
| `mxTarget` | `<tenant>.mail.protection.office365.us` | `<tenant>.mail.protection.office365.us` | copy it from the admin center |
| `autodiscoverTarget` | `autodiscover.office365.us` | `autodiscover-dod.office365.us` | copy it from the admin center |
| `sipFederationTarget` | `sipfed.online.gov.skypeforbusiness.us` | `sipfed.online.dod.skypeforbusiness.us` | copy it from the admin center |
| `mdmEnrollmentTarget` | `enterpriseenrollment-s.manage.microsoft.us` | `enterpriseenrollment-s.manage.microsoft.us` | `enterpriseenrollment-s.manage.microsoftonline.cn` |
| `mdmRegistrationTarget` | `enterpriseregistration.windows.net` | `enterpriseregistration.windows.net` | not documented |

Microsoft documents the device registration record for GCC High and DoD only,
so 21Vianet has no target to set.

Microsoft does not document DKIM targets for these clouds. Read the values from
the Microsoft 365 admin center or from `Get-DkimSigningConfig` and set
`dkimSelector1Target` and `dkimSelector2Target`, or set `dkim: false`.

Microsoft 365 operated by 21Vianet needs one more record that the builder does
not create: a [`CNAME`](CNAME.md) at `msoid` pointing to
`clientconfig.partner.microsoftonline-p.net.cn.`. It belongs to that cloud
alone; in any other cloud an `msoid` record keeps users from activating their
license.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  M365_BUILDER("example.com", {
      mxTarget: "contoso.mail.protection.office365.us",
      autodiscoverTarget: "autodiscover.office365.us",
      dkim: false, // Microsoft does not document DKIM targets for GCC High.
      skypeForBusiness: true,
      sipFederationTarget: "sipfed.online.gov.skypeforbusiness.us",
      mdm: true,
      mdmEnrollmentTarget: "enterpriseenrollment-s.manage.microsoft.us",
  }),
);
```
{% endcode %}

## TTL

The TTL is set with [`TTL`](../record-modifiers/TTL.md) as a further argument to
the call. It applies to every record the builder creates, and it is the only
record modifier the call takes.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  M365_BUILDER("example.com", {
      initialDomain: "contoso.onmicrosoft.com",
  }, TTL("1h")),
);
```
{% endcode %}

Microsoft documents a TTL of 3600 seconds for the `MX` record and at least 3600
seconds for the two DKIM `CNAME` records, where a shorter TTL causes
`dkim=temperror` results. Without a `TTL()` the zone's
[`DefaultTTL`](DefaultTTL.md) applies, which is 300 seconds unless the
`dnsconfig.js` sets another one.

## Complete example

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  M365_BUILDER("example.com", {
      // Both values come from the Microsoft 365 admin center; see the DKIM
      // section for the older format that `initialDomain` derives.
      dkimSelector1Target: "selector1-example-com._domainkey.contoso.n-v1.dkim.mail.microsoft",
      dkimSelector2Target: "selector2-example-com._domainkey.contoso.n-v1.dkim.mail.microsoft",
      verificationToken: "MS=ms12345678", // Remove this once the domain is verified.
      skypeForBusiness: true,
      mdm: true,
  }, TTL("1h")),

  SPF_BUILDER({
    label: "@",
    parts: [
      "v=spf1",
      "include:spf.protection.outlook.com", // Microsoft 365
      "-all",
    ],
  }),

  DMARC_BUILDER({
    policy: "reject",
    rua: [
      "mailto:dmarc-reports@example.com",
    ],
  }),
);
```
{% endcode %}

## Migrating from earlier versions

| Change | What to do |
|---|---|
| `label` now applies to the Autodiscover, DKIM, SIP federation and device management records, and the domain token is derived from `label` plus the zone name. | Nothing, if the records were already correct. Otherwise review the diff before pushing. |
| The first parameter must be the name the records are created under, unless `domainGUID` is set, or no target is derived from it. | Set `label`, or pass the name of the zone, or set `domainGUID`. |
| `skypeForBusiness` now creates only the `_sipfederationtls._tcp` record. The `lyncdiscover`, `sip` and `_sip._tls` records are gone; two of their targets no longer resolve. | Nothing. Add the records with `CNAME` and `SRV` if you still need them. |
| `mdm` now uses `enterpriseenrollment-s.manage.microsoft.com.`, which Microsoft calls the preferred name. | Nothing. Set `mdmEnrollmentTarget: "enterpriseenrollment.manage.microsoft.com."` to keep the old value. |
| Unknown option names, wrong types, empty strings, malformed values and more than two arguments are errors. | Fix the spelling or the value named in the error message. |
| Nine option names are new: `mxPriority`, `mxTarget`, `autodiscoverTarget`, `dkimSelector1Target`, `dkimSelector2Target`, `sipFederationTarget`, `mdmEnrollmentTarget`, `mdmRegistrationTarget` and `verificationToken`. A key of one of those names used to be ignored and now takes effect. | Review the diff before pushing if the options object carries any of them. |
| Target options and `initialDomain` must be a full host name. A value that is not fully qualified is an error instead of having the zone name appended to it. | Add the missing part of the name. |
| An internationalized domain name needs `domainGUID`, because its punycode form contains a dash. It used to produce a target built by encoding the unicode form, for example `xn--mnchen-de-q9a.mail.protection.outlook.com.`. | Set `domainGUID` to the value the Microsoft 365 admin center shows. |
| `TTL()` on the call now applies instead of being ignored. | Nothing, unless a stray `TTL()` was left in the call. |
| The options object is no longer modified. Sharing one object between several `D()` blocks used to give every later domain the first domain's MX and DKIM targets. | Nothing, those records were wrong. Review the diff before pushing. |
| A trailing dot in the domain name or in `initialDomain` is removed instead of ending up in the middle of a target. | Nothing, those targets were wrong. |

## References

- [External Domain Name System records for Microsoft 365](https://learn.microsoft.com/en-us/microsoft-365/enterprise/external-domain-name-system-records?view=o365-worldwide).
  The records of the worldwide cloud. Change the `view` parameter for GCC High,
  DoD and 21Vianet.
- [How to use DKIM for email in your custom domain](https://learn.microsoft.com/en-us/defender-office-365/email-authentication-dkim-configure).
  The DKIM targets, the format used for domains created from May 2025 onwards,
  and the TTL.
- [Get-DkimSigningConfig](https://learn.microsoft.com/en-us/powershell/module/exchange/get-dkimsigningconfig).
  Reads the two DKIM targets Microsoft assigned to a domain.
- [How SMTP DANE works](https://learn.microsoft.com/en-us/purview/how-smtp-dane-works)
  and [Enable-DnssecForVerifiedDomain](https://learn.microsoft.com/en-us/powershell/module/exchange/enable-dnssecforverifieddomain).
  The `MX` target of a tenant switched to DANE and DNSSEC.
- [Enable autodiscovery of Intune enrollment server](https://learn.microsoft.com/en-us/intune/device-enrollment/windows/create-cname-autodiscovery).
  The `enterpriseenrollment` record, and why `enterpriseenrollment-s` is the
  preferred name.
